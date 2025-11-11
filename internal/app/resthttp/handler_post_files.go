package resthttp

import (
	"encoding/json"
	"net/http"

	"github.com/sir_venger/s3_lite/pkg/httperrors"
)

type postFilesResp struct {
	FileID string `json:"file_id"`
	Size   int64  `json:"size"`
	Parts  int    `json:"parts"`
}

func (s *Server) postFiles(w http.ResponseWriter, r *http.Request) {
	res, err := s.FilesService.UploadWhole(r.Context(), r.Body, r.ContentLength)
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
