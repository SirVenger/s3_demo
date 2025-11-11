package resthttp

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sir_venger/s3_lite/pkg/httperrors"
)

type postFilesResp struct {
	FileID string `json:"file_id"`
	Size   int64  `json:"size"`
	Parts  int    `json:"parts"`
}

func (s *Server) postFiles(w http.ResponseWriter, r *http.Request) {
	filename := extractFileName(r)

	res, err := s.FilesService.UploadWhole(r.Context(), r.Body, r.ContentLength, filename)
	if err != nil {
		httperrors.Write(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(postFilesResp{
		FileID: res.FileID,
		Size:   res.Size,
		Parts:  res.Parts,
	})
}

func extractFileName(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-File-Name")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("X-Filename")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("filename")); v != "" {
		return v
	}
	return ""
}
