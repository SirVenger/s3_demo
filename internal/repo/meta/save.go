package meta

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/sir_venger/s3_lite/internal/models"
)

// Save записывает (или обновляет) описание файла.
func (s *PGStore) Save(ctx context.Context, file models.File) error {
	if file.Parts == nil {
		file.Parts = make(map[int]models.Part)
	}

	// Подготовка данных
	partsJSON, err := json.Marshal(file.Parts)
	if err != nil {
		return fmt.Errorf("marshal parts: %w", err)
	}

	sqlStr, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert(filesMetaTable).
		Columns("id", "file_name", "total_parts", "size", "parts").
		Values(file.ID, file.Name, file.TotalParts, file.Size, partsJSON).
		Suffix(`
					ON CONFLICT (id) DO UPDATE
					SET file_name   = EXCLUDED.file_name,
						total_parts = EXCLUDED.total_parts,
						size        = EXCLUDED.size,
						parts       = EXCLUDED.parts`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert sql: %w", err)
	}

	// Выполнение UPSERT'а
	if _, err := s.pool.Exec(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("exec upsert: %w", err)
	}

	return nil
}
