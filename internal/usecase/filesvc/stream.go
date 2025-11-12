package filesvc

import (
	"context"
	"errors"
	"io"

	"github.com/sir_venger/s3_lite/internal/models"
	"golang.org/x/sync/errgroup"
)

// Stream читает данные по частям из стораджей и транслирует клиенту.
func (s *Files) Stream(ctx context.Context, fileID string, w io.Writer) error {
	file, err := s.MetaStorage.Get(ctx, fileID)
	if err != nil {
		return err
	}
	if file.TotalParts == 0 {
		return nil
	}

	// Быстрый чек на «дырки»
	for i := 0; i < file.TotalParts; i++ {
		if _, ok := file.Parts[i]; !ok {
			return models.ErrIncomplete
		}
	}

	// errgroup с отменяемым контекстом
	eg, egCtx := errgroup.WithContext(ctx)

	// Семафор конкуренции
	sem := make(chan struct{}, file.TotalParts)

	// Для каждого индекса — свой pipe
	type pipePair struct {
		r *io.PipeReader
		w *io.PipeWriter
	}
	pipes := make([]pipePair, file.TotalParts)

	// Поднимаем загрузчики: каждый пишет свою часть в pipeWriter.
	for idx := 0; idx < file.TotalParts; idx++ {
		idx := idx
		part := file.Parts[idx]

		pr, pw := io.Pipe()
		pipes[idx] = pipePair{r: pr, w: pw}

		eg.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-egCtx.Done():
				pw.CloseWithError(egCtx.Err())
				return egCtx.Err()
			}
			defer func() { <-sem }()

			rc, err := s.StorageCli.GetPart(egCtx, part.Storage, file.ID, idx)
			if err != nil {
				_ = pw.CloseWithError(err)
				return nil
			}

			// ВНИМАНИЕ: закрываем rc после копирования
			_, copyErr := io.Copy(pw, rc)
			_ = rc.Close()
			_ = pw.CloseWithError(copyErr) // пробрасываем ошибку (или nil) на читательский конец
			return nil
		})
	}

	// Писатель: читает pipe'ы строго по порядку и пишет в w.
	for idx := 0; idx < file.TotalParts; idx++ {
		if _, err = io.Copy(w, pipes[idx].r); err != nil {
			_ = pipes[idx].r.CloseWithError(err)

			for j := idx + 1; j < file.TotalParts; j++ {
				err = pipes[j].r.Close()
				if err != nil {
					return err
				}
			}

			return eg.Wait()
		}

		err = pipes[idx].r.Close()
		if err != nil {
			return err
		}
	}

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
