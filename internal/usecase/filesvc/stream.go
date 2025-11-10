package filesvc

import (
	"context"
	"io"

	"github.com/yourname/storage_lite/internal/models"
)

// Stream читает данные по частям из стораджей и транслирует клиенту.
func (s *Files) Stream(ctx context.Context, fileID string, w io.Writer) error {
	file, err := s.MetaStorage.Get(fileID)
	if err != nil {
		return err
	}

	for idx := 0; idx < file.TotalParts; idx++ {
		part, ok := file.Parts[idx]
		if !ok {
			return models.ErrIncomplete
		}

		reader, err := s.StorageCli.GetPart(ctx, part.Storage, file.ID, idx)
		if err != nil {
			return err
		}

		if _, copyErr := io.Copy(w, reader); copyErr != nil {
			reader.Close()
			return copyErr
		}

		reader.Close()
	}

	return nil
}
