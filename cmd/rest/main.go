package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sir_venger/s3_lite/internal/app/resthttp"
	"github.com/sir_venger/s3_lite/internal/config"
)

// main инициализирует REST HTTP-сервис и обеспечивает корректное завершение по сигналу.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	handler, _, err := resthttp.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: handler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Сценарий graceful shutdown при получении SIGTERM/SIGINT.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("REST shutdown error: %v", err)
		}
	}()

	log.Printf("REST listening on %s", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("STORAGE final shutdown error: %v", err)
	}
}
