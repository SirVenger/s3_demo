package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/sir_venger/s3_lite/internal/config"
	meta "github.com/sir_venger/s3_lite/internal/repo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	dsn := strings.TrimSpace(cfg.MetaDSN)
	if dsn == "" {
		log.Fatal("meta_dsn is not configured")
	}
	if strings.HasPrefix(dsn, "memory://") {
		log.Println("memory meta store selected, skipping migrations")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := meta.ApplyMigrations(ctx, dsn); err != nil {
		log.Fatal(err)
	}

	log.Println("migrations applied")
}
