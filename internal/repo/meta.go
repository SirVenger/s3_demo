package meta

import (
	"sync"

	"github.com/sir_venger/s3_lite/internal/models"
)

// MemoryStore хранит метаданные только в оперативной памяти; удобно для тестов.
type MemoryStore struct {
	mu    sync.RWMutex
	files map[string]models.File
}

// NewMemoryStore создаёт пустое in-memory хранилище.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{files: map[string]models.File{}}
}

// Get возвращает метаданные файла по id или ошибку, если файл не найден.
func (s *MemoryStore) Get(id string) (models.File, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fr, ok := s.files[id]
	if !ok {
		return models.File{}, models.ErrNotFound
	}
	return fr.Clone(), nil
}

// Save записывает (или обновляет) метаданные файла целиком.
func (s *MemoryStore) Save(fr models.File) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fr.Parts == nil {
		fr.Parts = map[int]models.Part{}
	}
	s.files[fr.ID] = fr
	return nil
}
