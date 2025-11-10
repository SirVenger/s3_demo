package storagehttp

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const manualGCTTL = 24 * time.Hour

// gcOnce вручную запускает сбор старых незавершённых директорий.
func (a *Server) gcOnce(w http.ResponseWriter, _ *http.Request) {
	_ = sweepOnce(a.dataDir, manualGCTTL)
	w.WriteHeader(http.StatusNoContent)
}

// StartGC стартует периодическую очистку каталога.
func StartGC(root string, ttl time.Duration, every time.Duration) func() {
	if every <= 0 || ttl <= 0 {
		return func() {}
	}

	ticker := time.NewTicker(every)
	stop := make(chan struct{})
	var once sync.Once
	go func() {
		for {
			select {
			case <-ticker.C:
				_ = sweepOnce(root, ttl)
			case <-stop:
				ticker.Stop()
				return
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(stop)
		})
	}
}

// sweepOnce удаляет каталоги, у которых meta.json устарел и содержит неполные данные
func sweepOnce(root string, ttl time.Duration) error {
	now := time.Now()
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		pdir := filepath.Join(root, e.Name())
		metaPath := filepath.Join(pdir, metaFileName)
		fi, err := os.Stat(metaPath)
		if err != nil {
			continue
		}

		if now.Sub(fi.ModTime()) < ttl {
			continue
		}

		fm, err := readMeta(metaPath)
		if err != nil {
			continue
		}

		if len(fm.Parts) < fm.TotalParts {
			_ = os.RemoveAll(pdir)
		}
	}

	return nil
}
