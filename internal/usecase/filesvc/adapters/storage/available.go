package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

var healthHTTPClient = &http.Client{Timeout: 2 * time.Second}

// HealthAdapter определяет готовность стораджей по их health-эндпоинтам.
type HealthAdapter struct {
	MaxStorageLoadBytes int64
}

// NewHealthAdapter инициализирует адаптер доступности.
func NewHealthAdapter(maxLoad int64) *HealthAdapter {
	return &HealthAdapter{
		MaxStorageLoadBytes: maxLoad,
	}
}

// Available возвращает отсортированный список готовых стораджей.
func (a *HealthAdapter) Available(ctx context.Context, storages []string) []string {
	if len(storages) == 0 {
		return nil
	}

	type candidate struct {
		base string
		load int64
	}

	ready := make([]candidate, 0, len(storages))
	for _, base := range storages {
		info, err := fetchStorageHealth(ctx, base)
		if err != nil || !info.OK {
			continue
		}
		if !a.loadAcceptable(info.TotalBytes) {
			continue
		}
		ready = append(ready, candidate{
			base: base,
			load: info.TotalBytes,
		})
	}

	sort.Slice(ready, func(i, j int) bool {
		return ready[i].load < ready[j].load
	})

	result := make([]string, len(ready))
	for i, c := range ready {
		result[i] = c.base
	}
	return result
}

func (a *HealthAdapter) loadAcceptable(load int64) bool {
	if a.MaxStorageLoadBytes <= 0 {
		return true
	}
	return load <= a.MaxStorageLoadBytes
}

type storageHealth struct {
	OK         bool  `json:"ok"`
	FreeBytes  int64 `json:"free_bytes"`
	TotalBytes int64 `json:"total_bytes"`
}

func fetchStorageHealth(ctx context.Context, base string) (payload storageHealth, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL(base), nil)
	if err != nil {
		return storageHealth{}, err
	}

	resp, err := healthHTTPClient.Do(req)
	if err != nil {
		return storageHealth{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return storageHealth{}, fmt.Errorf("health check failed: %s", resp.Status)
	}

	if err = json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return storageHealth{}, err
	}

	return payload, nil
}

func healthURL(base string) string {
	return strings.TrimRight(base, "/") + "/health"
}
