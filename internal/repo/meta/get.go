package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/sir_venger/s3_lite/internal/models"
)

// Get возвращает описание файла по его идентификатору.
func (s *PGStore) Get(ctx context.Context, id string) (models.File, error) {
	if strings.TrimSpace(id) == "" {
		return models.File{}, fmt.Errorf("file id is empty")
	}

	// COALESCE(parts, '{}') — чтобы гарантированно получить валидный JSON для Unmarshal, это для себя комментарий
	sqlStr, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select(
			"file_name",
			"total_parts",
			"size",
			"COALESCE(parts, '{}'::jsonb) AS parts",
		).
		From(filesMetaTable).
		Where(sq.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return models.File{}, fmt.Errorf("build select: %w", err)
	}

	var (
		name       string
		totalParts int
		size       int64
		partsRaw   []byte
	)

	if err = s.pool.QueryRow(ctx, sqlStr, args...).Scan(&name, &totalParts, &size, &partsRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.File{}, models.ErrNotFound
		}
		return models.File{}, fmt.Errorf("scan file row: %w", err)
	}

	var parts map[int]models.Part
	if err := json.Unmarshal(partsRaw, &parts); err != nil {
		return models.File{}, fmt.Errorf("unmarshal parts: %w", err)
	}
	if parts == nil {
		parts = make(map[int]models.Part)
	}

	return models.File{
		ID:         id,
		Name:       name,
		Size:       size,
		TotalParts: totalParts,
		Parts:      parts,
	}.Clone(), nil
}
