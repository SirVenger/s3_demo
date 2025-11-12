package filesvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
func (s *Files) UploadWhole(ctx context.Context, r io.Reader, size int64, name string) (models.UploadResult, error) {
	if size < 0 {
		return models.UploadResult{}, fmt.Errorf("content length is required")
	}

	plan := determineParts(size, s.Parts)
	storages, err := s.Router.Allocate(ctx, plan.Total)
	if err != nil {
		return models.UploadResult{}, err
	}

	fileID := uuid.NewString()
	file := models.File{
		ID:         fileID,
		Name:       strings.TrimSpace(name),
		Size:       size,
		TotalParts: plan.Total,
		Parts:      make(map[int]models.Part, plan.Total),
	}

	sem := make(chan struct{}, plan.Total)

	eg, egCtx := errgroup.WithContext(ctx)
	var mu sync.Mutex

	remaining := size
	for idx := 0; idx < plan.Total; idx++ {
		if err = egCtx.Err(); err != nil {
			return models.UploadResult{}, err
		}

		partSize := min(plan.Size, remaining)
		pr, pw := io.Pipe()

		select {
		case sem <- struct{}{}:
		case <-egCtx.Done():
			_ = pw.CloseWithError(egCtx.Err())
			return models.UploadResult{}, egCtx.Err()
		}

		// Воркера запускаем сразу: он будет читать из пайпа и заливать часть.
		storage := storages[idx]
		eg.Go(func(i int, rd *io.PipeReader, st string, expected int64) func() error {
			return func() error {
				defer func() { <-sem }()
				defer rd.Close()
				req := storageclient.PutPartRequest{
					FileID:     fileID,
					Index:      i,
					Reader:     rd,
					Size:       expected,
					TotalParts: plan.Total,
				}
				if err = s.StorageCli.PutPart(egCtx, st, req); err != nil {
					return err
				}
				return nil
			}
		}(idx, pr, storage, partSize))

		// Продюсер: читаем кусок из входного r и пишем в PipeWriter,
		// одновременно считаем SHA-256 без дополнительного буфера.
		hasher := sha256.New()
		limited := &io.LimitedReader{R: r, N: partSize}
		tee := io.TeeReader(limited, hasher)
		n, copyErr := io.Copy(pw, tee)
		closeErr := pw.CloseWithError(copyErr) // проброс для воркера

		if copyErr != nil {
			_ = eg.Wait()

			if errors.Is(copyErr, io.ErrClosedPipe) && egCtx.Err() != nil {
				return models.UploadResult{}, egCtx.Err()
			}

			return models.UploadResult{}, copyErr
		}
		if closeErr != nil {
			_ = eg.Wait()
			return models.UploadResult{}, closeErr
		}
		if n != partSize {
			_ = eg.Wait()
			return models.UploadResult{}, fmt.Errorf("unexpected part length: want %d, got %d", partSize, n)
		}

		sha := hex.EncodeToString(hasher.Sum(nil))

		mu.Lock()
		file.Parts[idx] = models.Part{
			Index:   idx,
			Size:    n,
			Sha256:  sha,
			Storage: storage,
		}
		mu.Unlock()

		remaining -= n
	}

	if err = eg.Wait(); err != nil {
		return models.UploadResult{}, err
	}
	if remaining != 0 {
		return models.UploadResult{}, fmt.Errorf("incomplete upload: %d bytes left", remaining)
	}

	if err = s.MetaStorage.Save(ctx, file); err != nil {
		return models.UploadResult{}, err
	}

	return models.UploadResult{FileID: fileID, Size: size, Parts: plan.Total}, nil
}

// determineParts вычисляет оптимальное число частей и размер каждой.
func determineParts(length int64, desired int) models.ChunkPlan {
	if desired <= 0 {
		desired = 1
	}
	if length <= 0 {
		return models.ChunkPlan{Total: 1, Size: 0}
	}

	chunkSize := int64(math.Ceil(float64(length) / float64(desired)))
	if chunkSize <= 0 {
		chunkSize = 1
	}

	totalParts := int((length + chunkSize - 1) / chunkSize)
	if totalParts <= 0 {
		totalParts = 1
	}

	return models.ChunkPlan{
		Total: totalParts,
		Size:  chunkSize,
	}
}
