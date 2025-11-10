package filesvc

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/yourname/storage_lite/internal/models"
)

// StorageAdapter описывает источник знаний о доступности стораджей.
type StorageAdapter interface {
	Available(ctx context.Context, storages []string) []string
}

// Router отвечает за выбор стораджей для записи файлов.
type Router struct {
	mu             sync.Mutex
	configured     []string
	next           int
	StorageAdapter StorageAdapter
}

// NewRouter создаёт маршрутизатор с адаптером доступности.
func NewRouter(adapter StorageAdapter) *Router {
	return &Router{StorageAdapter: adapter}
}

// Set заменяет список стораджей на новый.
func (r *Router) Set(storages []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configured = append([]string{}, storages...)
	r.next = 0
}

// Add добавляет новые стораджи, игнорируя дубликаты и пустые значения.
func (r *Router) Add(storages ...string) {
	if len(storages) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	known := make(map[string]struct{}, len(r.configured))
	for _, s := range r.configured {
		known[s] = struct{}{}
	}

	for _, storage := range storages {
		storage = strings.TrimSpace(storage)
		if storage == "" {
			continue
		}

		if _, exists := known[storage]; exists {
			continue
		}

		r.configured = append(r.configured, storage)
		known[storage] = struct{}{}
	}
}

// Allocate возвращает список стораджей длиной count.
func (r *Router) Allocate(ctx context.Context, count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	r.mu.Lock()
	if len(r.configured) == 0 {
		r.mu.Unlock()
		return nil, fmt.Errorf("no storages configured")
	}
	snapshot := append([]string{}, r.configured...)
	r.mu.Unlock()

	available := r.StorageAdapter.Available(ctx, snapshot)
	if len(available) == 0 {
		available = snapshot
	}

	if len(snapshot) == 0 && len(available) == 0 {
		return nil, models.ErrNoStorage
	}

	r.mu.Lock()
	start := r.next % len(available)
	r.next = (start + count) % len(available)
	r.mu.Unlock()

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = available[(start+i)%len(available)]
	}

	return result, nil
}
