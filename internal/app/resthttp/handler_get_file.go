package resthttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/yourname/storage_lite/pkg/httperrors"
)

func (s *Server) getFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.FilesService.Stream(r.Context(), id, w); err != nil {
		httperrors.Write(w, err)
		return
	}
}
