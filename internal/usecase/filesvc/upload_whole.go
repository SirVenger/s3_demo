package filesvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/sir_venger/s3_lite/internal/models"
	"github.com/sir_venger/s3_lite/pkg/storageclient"
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

	remaining := size
	for idx := 0; idx < plan.Total; idx++ {
		if ctx.Err() != nil {
			return models.UploadResult{}, ctx.Err()
		}

		partSize := min(plan.Size, remaining)
		limited := &io.LimitedReader{R: r, N: partSize}
		hasher := sha256.New()
		reader := io.TeeReader(limited, hasher)
		req := storageclient.PutPartRequest{
			FileID:     fileID,
			Index:      idx,
			Reader:     reader,
			Size:       partSize,
			TotalParts: plan.Total,
		}

		storage := storages[idx]
		if err = s.StorageCli.PutPart(ctx, storage, req); err != nil {
			return models.UploadResult{}, err
		}

		written := partSize - limited.N
		if written != partSize {
			return models.UploadResult{}, fmt.Errorf("unexpected part length: want %d, got %d", partSize, written)
		}
		sha := hex.EncodeToString(hasher.Sum(nil))
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

	return models.ChunkPlan{
		Total: desired,
		Size:  chunkSize,
	}
}
