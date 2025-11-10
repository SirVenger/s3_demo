package meta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/yourname/storage_lite/internal/models"
)

type Store struct {
	path  string
	mu    sync.RWMutex
	files map[string]models.File
}

// Open открывает (или создаёт) хранилище метаданных на указанном пути.
func Open(path string) (*Store, error) {
	s := &Store{path: path, files: map[string]models.File{}}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &s.files)
	}
	return s, nil
}

// Get возвращает метаданные файла по id или ошибку, если файл не найден.
func (s *Store) Get(id string) (models.File, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fr, ok := s.files[id]
	if !ok {
		return models.File{}, models.ErrNotFound
	}
	return fr.Clone(), nil
}

// Save записывает (или обновляет) метаданные файла целиком.
func (s *Store) Save(fr models.File) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fr.Parts == nil {
		fr.Parts = map[int]models.Part{}
	}
	s.files[fr.ID] = fr
	return s.Persist()
}

// Persist сохраняет текущее состояние в файл.
func (s *Store) Persist() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s.files, "", "  ")
	return os.WriteFile(s.path, b, 0o644)
}
