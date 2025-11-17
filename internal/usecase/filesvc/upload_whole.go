package filesvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/sir_venger/s3_lite/internal/models"
	"github.com/sir_venger/s3_lite/pkg/storageclient"
	"golang.org/x/sync/errgroup"
)

// UploadWhole читает поток постранично, делит на части и распределяет их по стораджам.
func (s *Files) UploadWhole(ctx context.Context, src io.Reader, size int64, name string) (models.UploadResult, error) {
	if size < 0 {
		// Контент без Content-Length не можем разбить на части заранее.
		return models.UploadResult{}, fmt.Errorf("content length is required")
	}

	// Планируем как делить файл, чтобы не превышать лимит параллельных частей.
	plan := determineParts(size, s.Parts)
	// Router отвечает, какие сториджи возьмут конкретные части файла.
	storages, err := s.Router.Allocate(ctx, plan.Total)
	if err != nil {
		return models.UploadResult{}, err
	}

	// Заготовка метаданных, которые позже уйдут в MetaStorage.
	fileID := uuid.NewString()
	file := models.File{
		ID:         fileID,
		Name:       strings.TrimSpace(name),
		Size:       size,
		TotalParts: plan.Total,
		Parts:      make(map[int]models.Part, plan.Total),
	}

	// Ограничиваем число одновременных воркеров количеством частей.
	workerSlots := make(chan struct{}, plan.Total)

	// errgroup спрячет синхронизацию и позволит отменять пайплайн целиком.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	group, groupCtx := errgroup.WithContext(ctx)
	fail := func(err error) (models.UploadResult, error) {
		cancel()
		_ = group.Wait()
		if err == io.ErrClosedPipe && groupCtx.Err() != nil {
			return models.UploadResult{}, groupCtx.Err()
		}
		return models.UploadResult{}, err
	}
	var mu sync.Mutex

	// remaining отслеживает, сколько байт ещё нужно прочитать из r.
	remaining := size
	// Бежим по индексам частей и для каждой поднимаем пайп и воркер.
	for partIdx := 0; partIdx < plan.Total; partIdx++ {
		if err := groupCtx.Err(); err != nil {
			// Если контекст уже отменён, прекращаем чтение новых частей.
			return fail(err)
		}

		// Последняя часть может быть меньше планового размера, поэтому ограничиваем остатком.
		partSize := min(plan.Size, remaining)
		// Pipe даёт поток между продюсером, читающим клиентский upload, и воркером.
		partReader, partWriter := io.Pipe()

		// Блокируемся до освобождения семафора или выходим, если контекст отменён.
		select {
		case workerSlots <- struct{}{}:
		case <-groupCtx.Done():
			_ = partWriter.CloseWithError(groupCtx.Err())
			return fail(groupCtx.Err())
		}

		// Воркера запускаем сразу: он будет читать из пайпа и заливать часть.
		storageURL := storages[partIdx]
		group.Go(func(index int, rd *io.PipeReader, base string, expected int64) func() error {
			return func() error {
				defer func() { <-workerSlots }()
				defer rd.Close()
				req := storageclient.PutPartRequest{
					FileID:     fileID,
					Index:      index,
					Reader:     rd,
					Size:       expected,
					TotalParts: plan.Total,
				}
				if err := s.StorageCli.PutPart(groupCtx, base, req); err != nil {
					cancel()
					return err
				}
				return nil
			}
		}(partIdx, partReader, storageURL, partSize))

		// Продюсер: читаем кусок из входного r и пишем в PipeWriter,
		// одновременно считаем SHA-256 без дополнительного буфера.
		hasher := sha256.New()
		limitedSrc := &io.LimitedReader{R: src, N: partSize}
		hashedStream := io.TeeReader(limitedSrc, hasher)

		// copyErr повлияет и на закрытие пайпа
		written, copyErr := io.Copy(partWriter, hashedStream)
		closeErr := partWriter.CloseWithError(copyErr) // проброс для воркера

		if copyErr != nil {
			// Отменим группу и дождёмся остановки
			return fail(copyErr)
		}
		if closeErr != nil {
			return fail(closeErr)
		}
		if written != partSize {
			// Поток закончился раньше — сигнализируем об ошибке, чтобы клиент перезалил файл.
			return fail(fmt.Errorf("unexpected part length: want %d, got %d", partSize, written))
		}

		// Храним хеш части для последующей валидации при сборке.
		sha := hex.EncodeToString(hasher.Sum(nil))
		// Пишем часть в карту потокобезопасно, чтобы не гоняться за race.
		mu.Lock()
		file.Parts[partIdx] = models.Part{
			Index:   partIdx,
			Size:    written,
			Sha256:  sha,
			Storage: storageURL,
		}
		mu.Unlock()

		// Урезаем остаток, чтобы следующая итерация знала, сколько байт ещё нужно.
		remaining -= n
	}

	// Убедимся, что все воркеры завершились без ошибок.
	if err := group.Wait(); err != nil {
		return models.UploadResult{}, err
	}
	// Если осталось что-то непрочитанным — значит поток закончился раньше ожидаемого.
	if remaining != 0 {
		return models.UploadResult{}, fmt.Errorf("incomplete upload: %d bytes left", remaining)
	}

	// После успешной раздачи частей фиксируем запись о файле.
	if err := s.MetaStorage.Save(ctx, file); err != nil {
		return models.UploadResult{}, err
	}

	// Отдаём клиенту идентификатор и итоговую статистику.
	return models.UploadResult{FileID: fileID, Size: size, Parts: plan.Total}, nil
}

// determineParts вычисляет оптимальное число частей и размер каждой.
func determineParts(length int64, desired int) models.ChunkPlan {
	if desired <= 0 {
		// Минимум одна часть, иначе деление смысла не имеет.
		desired = 1
	}
	if length <= 0 {
		// Нулевой размер возвращаем как одну пустую часть, чтобы код выше не ломался.
		return models.ChunkPlan{Total: 1, Size: 0}
	}

	// Размер части вычисляем через ceil, чтобы покрыть весь файл.
	chunkSize := int64(math.Ceil(float64(length) / float64(desired)))
	if chunkSize <= 0 {
		chunkSize = 1
	}

	// Уточняем итоговое количество частей (ceil, но в целых числах), чтобы не потерять хвост.
	totalParts := int((length + chunkSize - 1) / chunkSize)
	if totalParts <= 0 {
		totalParts = 1
	}

	return models.ChunkPlan{
		Total: totalParts,
		Size:  chunkSize,
	}
}
