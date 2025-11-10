package storagehttp

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type partRequest struct {
	fileID string
	idx    int
	dir    string
	part   string
	meta   string
}

func (a *Server) requirePartRequest(w http.ResponseWriter, r *http.Request) (*partRequest, bool) {
	req, err := newPartRequest(a.dataDir, r)
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}

	return req, true
}

func newPartRequest(root string, r *http.Request) (*partRequest, error) {
	fileID := chi.URLParam(r, "fileID")
	idxStr := chi.URLParam(r, "idx")
	if fileID == "" || idxStr == "" {
		return nil, fmt.Errorf("invalid path")
	}

	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return nil, fmt.Errorf("invalid part index: %w", err)
	}
	if idx < 0 {
		return nil, fmt.Errorf("invalid part index: must be non-negative")
	}

	dir := filepath.Join(root, fileID)

	return &partRequest{
		fileID: fileID,
		idx:    idx,
		dir:    dir,
		part:   filepath.Join(dir, fmt.Sprintf(partFilenameFormat, idx)),
		meta:   filepath.Join(dir, metaFileName),
	}, nil
}
