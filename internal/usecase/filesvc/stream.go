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

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// errgroup с отменяемым контекстом
	eg, egCtx := errgroup.WithContext(streamCtx)

	// Семафор конкуренции
	sem := make(chan struct{}, file.TotalParts)

	// Для каждого индекса — свой pipe
	type pipePair struct {
		r *io.PipeReader
	}
	pipes := make([]pipePair, file.TotalParts)

	// Поднимаем загрузчики: каждый пишет свою часть в pipeWriter.
	for idx := 0; idx < file.TotalParts; idx++ {
		idx := idx
		part := file.Parts[idx]

		pr, pw := io.Pipe()
		pipes[idx] = pipePair{r: pr}

		eg.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-egCtx.Done():
				_ = pw.CloseWithError(egCtx.Err())
				return egCtx.Err()
			}
			defer func() { <-sem }()

			rc, err := s.StorageCli.GetPart(egCtx, part.Storage, file.ID, idx)
			if err != nil {
				_ = pw.CloseWithError(err)
				return err
			}
			defer rc.Close()

			_, copyErr := io.Copy(pw, rc)
			closeErr := pw.CloseWithError(copyErr)
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			return nil
		})
	}

	// Писатель: читает pipe'ы строго по порядку и пишет в w.
	for idx := 0; idx < file.TotalParts; idx++ {
		reader := pipes[idx].r
		if _, err = io.Copy(w, reader); err != nil {
			cancel()
			_ = reader.CloseWithError(err)
			for j := idx + 1; j < file.TotalParts; j++ {
				_ = pipes[j].r.CloseWithError(err)
			}

			waitErr := eg.Wait()
			if waitErr != nil && !errors.Is(waitErr, context.Canceled) {
				return waitErr
			}
			return err
		}

		if err = reader.Close(); err != nil {
			cancel()
			for j := idx + 1; j < file.TotalParts; j++ {
				_ = pipes[j].r.CloseWithError(err)
			}

			waitErr := eg.Wait()
			if waitErr != nil && !errors.Is(waitErr, context.Canceled) {
				return waitErr
			}
			return err
		}
	}

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
