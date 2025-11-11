package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sir_venger/s3_lite/internal/models"
)

const (
	upsertMetaSQL = `
INSERT INTO files_meta(id, total_parts, payload)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE
SET total_parts = EXCLUDED.total_parts,
	payload = EXCLUDED.payload`

	selectMetaSQL = `
SELECT payload
FROM files_meta
WHERE id = $1`
)

// PGStore сохраняет метаданные в Postgres.
type PGStore struct {
	pool *pgxpool.Pool
}

// OpenPostgres создаёт подключение к Postgres и гарантирует наличие таблицы для файлов.
func OpenPostgres(ctx context.Context, dsn string) (*PGStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("meta dsn is empty")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &PGStore{pool: pool}, nil
}

// Get возвращает описание файла по его идентификатору.
func (s *PGStore) Get(id string) (models.File, error) {
	var payload []byte
	err := s.pool.QueryRow(context.Background(), selectMetaSQL, id).Scan(&payload)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.File{}, models.ErrNotFound
		}
		return models.File{}, err
	}

	var file models.File
	if err := json.Unmarshal(payload, &file); err != nil {
		return models.File{}, err
	}
	if file.ID == "" {
		file.ID = id
	}

	return file.Clone(), nil
}

// Save записывает (или обновляет) описание файла.
func (s *PGStore) Save(file models.File) error {
	if strings.TrimSpace(file.ID) == "" {
		return fmt.Errorf("file id is empty")
	}
	if file.Parts == nil {
		file.Parts = map[int]models.Part{}
	}

	payload, err := json.Marshal(file)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(
		context.Background(),
		upsertMetaSQL,
		file.ID,
		file.TotalParts,
		payload,
	)
	return err
}

// Close освобождает подключения пула.
func (s *PGStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}
