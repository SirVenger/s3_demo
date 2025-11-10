package filesvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/google/uuid"
	"github.com/yourname/storage_lite/internal/models"
	"github.com/yourname/storage_lite/pkg/storageclient"
)

const tempChunkPattern = "storage-lite-part-*"

// UploadWhole читает поток постранично, делит на части и распределяет их по стораджам.
func (s *Files) UploadWhole(ctx context.Context, r io.Reader, size int64) (models.UploadResult, error) {
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
		TotalParts: plan.Total,
		Parts:      make(map[int]models.Part, plan.Total),
	}

	remaining := size
	for idx := 0; idx < plan.Total; idx++ {
		if ctx.Err() != nil {
			return models.UploadResult{}, ctx.Err()
		}

		partSize := min(plan.Size, remaining)
		chunkFile, written, sha, err := writeChunkToTemp(r, partSize)
		if err != nil {
			return models.UploadResult{}, err
		}

		reader := io.NewSectionReader(chunkFile, 0, written)
		req := storageclient.PutPartRequest{
			FileID:     fileID,
			Index:      idx,
			Reader:     reader,
			Size:       written,
			Sha256:     sha,
			TotalParts: plan.Total,
		}

		storage := storages[idx]
		putErr := s.StorageCli.PutPart(ctx, storage, req)
		closeErr := chunkFile.Close()
		removeErr := os.Remove(chunkFile.Name())
		if putErr != nil {
			return models.UploadResult{}, putErr
		}
		if closeErr != nil {
			return models.UploadResult{}, closeErr
		}
		if removeErr != nil && !os.IsNotExist(removeErr) {
			return models.UploadResult{}, removeErr
		}

		file.Parts[idx] = models.Part{
			Index:   idx,
			Size:    written,
			Sha256:  sha,
			Storage: storage,
		}

		remaining -= written
	}

	if remaining != 0 {
		return models.UploadResult{}, fmt.Errorf("incomplete upload: %d bytes left", remaining)
	}

	if err := s.MetaStorage.Save(file); err != nil {
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

	return models.ChunkPlan{
		Total: desired,
		Size:  chunkSize,
	}
}

func writeChunkToTemp(r io.Reader, expected int64) (*os.File, int64, string, error) {
	tmp, err := os.CreateTemp("", tempChunkPattern)
	if err != nil {
		return nil, 0, "", err
	}

	hasher := sha256.New()
	writer := io.MultiWriter(tmp, hasher)
	written, err := io.CopyN(writer, r, expected)
	if err != nil {
		err = tmp.Close()
		if err != nil {
			return nil, 0, "", err
		}

		err = os.Remove(tmp.Name())
		if err != nil {
			return nil, 0, "", err
		}

		return nil, written, "", err
	}

	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		err = tmp.Close()
		if err != nil {
			return nil, 0, "", err
		}

		err = os.Remove(tmp.Name())
		if err != nil {
			return nil, 0, "", err
		}

		return nil, written, "", err
	}

	return tmp, written, hex.EncodeToString(hasher.Sum(nil)), nil
}
