package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/yourname/storage_lite/internal/app/storagehttp"
)

const (
	defaultStorageAddr   = ":8081"
	dataDirEnv           = "DATA_DIR"
	gcTTLHoursEnv        = "GC_TTL_HOURS"
	gcIntervalMinEnv     = "GC_INTERVAL_MIN"
	defaultDataDir       = "/data"
	defaultGCTTLHours    = 24
	defaultGCIntervalMin = 30
)

func main() {
	addr := flag.String("addr", defaultStorageAddr, "listen address")
	flag.Parse()

	dataDir := os.Getenv(dataDirEnv)
	if dataDir == "" {
		dataDir = defaultDataDir
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	h := storagehttp.New(dataDir)

	// Настраиваем фоновый GC по удалению незавершённых загрузок.
	gcTTLHours := envInt(gcTTLHoursEnv, defaultGCTTLHours)
	gcEveryMin := envInt(gcIntervalMinEnv, defaultGCIntervalMin)
	stopGC := storagehttp.StartGC(dataDir, time.Duration(gcTTLHours)*time.Hour, time.Duration(gcEveryMin)*time.Minute)
	defer stopGC()

	server := &http.Server{Addr: *addr, Handler: h}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
			log.Printf("STORAGE shutdown error: %v", err)
		}
	}()

	log.Printf("STORAGE listening on %s (DATA_DIR=%s, GC ttl=%dh, every=%dm)", *addr, dataDir, gcTTLHours, gcEveryMin)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		log.Printf("STORAGE final shutdown error: %v", err)
	}
}

// envInt возвращает целочисленное значение из переменной окружения либо дефолт.
func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
