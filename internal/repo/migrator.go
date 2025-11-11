package meta

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const migrationsDir = "migrations"

// ApplyMigrations запускает goose-миграции для Postgres, используя встроенные SQL файлы.
func ApplyMigrations(ctx context.Context, dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return fmt.Errorf("meta dsn is empty")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	goose.SetDialect("postgres")
	goose.SetBaseFS(migrationFiles)

	return goose.Up(db, migrationsDir)
}
