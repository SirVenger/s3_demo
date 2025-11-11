package meta

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore сохраняет метаданные в Postgres.
type PGStore struct {
	pool *pgxpool.Pool
}

const filesMetaTable = "files_meta"

// NewPGStore создаёт подключение к Postgres и гарантирует наличие таблицы для файлов.
func NewPGStore(ctx context.Context, dsn string) (*PGStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("meta dsn is empty")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &PGStore{
		pool: pool,
	}, nil
}

// Close освобождает подключения пула.
func (s *PGStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}
