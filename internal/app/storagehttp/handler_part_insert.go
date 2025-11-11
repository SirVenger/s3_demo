package storagehttp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/sir_venger/s3_lite/pkg/storageproto"
)

// insertPart принимает POST-запросы на запись новой части.
func (a *Server) insertPart(w http.ResponseWriter, r *http.Request) {
	req, ok := a.requirePartRequest(w, r)
	if !ok {
		return
	}
	a.writePart(w, r, req)
}

func (a *Server) writePart(w http.ResponseWriter, r *http.Request, req *partRequest) {
	if err := os.MkdirAll(req.dir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	expSha := r.Header.Get(storageproto.HeaderChecksum)
	size, err := parseContentLength(r.Header.Get("Content-Length"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f, err := os.Create(req.part)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	h := sha256.New()
	wrt := io.MultiWriter(f, h)
	n, err := io.Copy(wrt, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if size > 0 && n != size {
		http.Error(w, "size mismatch", http.StatusBadRequest)
		return
	}
	got := hex.EncodeToString(h.Sum(nil))
	if expSha != "" && got != expSha {
		http.Error(w, "sha256 mismatch", http.StatusConflict)
		return
	}

	totalPartsStr := r.Header.Get(storageproto.HeaderTotalParts)
	totalParts, err := strconv.Atoi(totalPartsStr)
	if err != nil || totalParts <= 0 {
		http.Error(w, "invalid total parts header", http.StatusBadRequest)
		return
	}

	if err = writeMeta(req.meta, req.fileID, req.idx, n, got, totalParts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func parseContentLength(value string) (int64, error) {
	if value == "" {
		return -1, nil
	}

	sz, err := strconv.ParseInt(value, 10, 64)
	if err != nil || sz < 0 {
		return 0, fmt.Errorf("invalid Content-Length header")
	}

	return sz, nil
}
