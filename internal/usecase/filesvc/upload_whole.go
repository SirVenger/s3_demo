package filesvc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"

	"github.com/google/uuid"
	"github.com/sir_venger/s3_lite/internal/models"
	"github.com/sir_venger/s3_lite/pkg/storageclient"
)

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
		chunk, written, sha, err := readChunk(r, partSize)
		if err != nil {
			return models.UploadResult{}, err
		}

		reader := bytes.NewReader(chunk)
		req := storageclient.PutPartRequest{
			FileID:     fileID,
			Index:      idx,
			Reader:     reader,
			Size:       written,
			Sha256:     sha,
			TotalParts: plan.Total,
		}

		storage := storages[idx]
		if err := s.StorageCli.PutPart(ctx, storage, req); err != nil {
			return models.UploadResult{}, err
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

func readChunk(r io.Reader, expected int64) ([]byte, int64, string, error) {
	if expected < 0 {
		return nil, 0, "", fmt.Errorf("invalid chunk size: %d", expected)
	}

	hasher := sha256.New()
	var buf bytes.Buffer
	if expected > 0 {
		if expected > int64(int(^uint(0)>>1)) {
			return nil, 0, "", fmt.Errorf("chunk size %d exceeds buffer limit", expected)
		}
		buf.Grow(int(expected))
	}

	writer := io.MultiWriter(&buf, hasher)
	written, err := io.CopyN(writer, r, expected)
	if err != nil {
		return nil, written, "", err
	}

	return buf.Bytes(), written, hex.EncodeToString(hasher.Sum(nil)), nil
}
