package storagehttp

import (
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/sir_venger/s3_lite/pkg/storageproto"
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

	info, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	size := info.Size()
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set(storageproto.HeaderPartSize, strconv.FormatInt(size, 10))
	w.Header().Set("Content-Type", "application/octet-stream")

	if _, err = io.Copy(w, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	return
}
