package storagehttp

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"path/filepath"
)

// healthStats — payload ответа /health.
type healthStats struct {
	OK         bool  `json:"ok"`
	FreeBytes  int64 `json:"free_bytes"`
	TotalBytes int64 `json:"total_bytes"`
}

// health возвращает агрегированную статистику по данным стоража.
func (a *Server) health(w http.ResponseWriter, r *http.Request) {
	var total int64
	// Проходим по всем файлам в dataDir и суммируем их размер для простого capacity-метрика.
	err := filepath.WalkDir(a.dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()

		return nil
	})

	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// У стораджа нет сложных метрик, поэтому отдаём только total и флаг OK.
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(healthStats{
		OK:         true,
		TotalBytes: total,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
