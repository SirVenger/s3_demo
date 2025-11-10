package storagehttp

import (
	"io"
	"net/http"
	"os"
)

// fetchPart обслуживает GET-запросы, возвращая содержимое части.
func (a *Server) fetchPart(w http.ResponseWriter, r *http.Request) {
	req, ok := a.requirePartRequest(w, r)
	if !ok {
		return
	}

	f, err := os.Open(req.part)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	if _, err = io.Copy(w, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	return
}
