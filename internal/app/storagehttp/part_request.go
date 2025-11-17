package storagehttp

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// partRequest содержит все вычисленные пути до части и её метаданных.
type partRequest struct {
	fileID string
	idx    int
	dir    string
	part   string
	meta   string
}

// requirePartRequest валидирует path-параметры и возвращает заполненную структуру.
func (a *Server) requirePartRequest(w http.ResponseWriter, r *http.Request) (*partRequest, bool) {
	req, err := newPartRequest(a.dataDir, r)
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}

	return req, true
}

// newPartRequest парсит идентификаторы из URL и рассчитывает пути на диске.
func newPartRequest(root string, r *http.Request) (*partRequest, error) {
	// fileID/idx берутся напрямую из path-параметров Chi.
	fileID := chi.URLParam(r, "fileID")
	idxStr := chi.URLParam(r, "idx")
	if fileID == "" || idxStr == "" {
		return nil, fmt.Errorf("invalid path")
	}

	// Индекс части приходит в десятиричном виде, отрицательные значения запрещены.
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return nil, fmt.Errorf("invalid part index: %w", err)
	}
	if idx < 0 {
		return nil, fmt.Errorf("invalid part index: must be non-negative")
	}

	// Каждому файлу соответствует собственная директория в dataDir.
	dir := filepath.Join(root, fileID)

	return &partRequest{
		fileID: fileID,
		idx:    idx,
		dir:    dir,
		part:   filepath.Join(dir, fmt.Sprintf(partFilenameFormat, idx)),
		meta:   filepath.Join(dir, metaFileName),
	}, nil
}
