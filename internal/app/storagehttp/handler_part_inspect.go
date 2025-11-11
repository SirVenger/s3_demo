package storagehttp

import (
	"net/http"
	"strconv"

	"github.com/sir_venger/s3_lite/pkg/storageproto"
)

// inspectPart отвечает на HEAD-запросы метаданными по части.
func (a *Server) inspectPart(w http.ResponseWriter, r *http.Request) {
	req, ok := a.requirePartRequest(w, r)
	if !ok {
		return
	}

	meta, err := readMeta(req.meta)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	part, ok := meta.Parts[req.idx]
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set(storageproto.HeaderPartSize, strconv.FormatInt(part.Size, 10))
	w.Header().Set(storageproto.HeaderChecksum, part.Sha256)
	w.WriteHeader(http.StatusOK)

	return
}
