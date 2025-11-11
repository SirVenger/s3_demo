package httperrors

import (
	"errors"
	"net/http"
	"strings"

	"github.com/sir_venger/s3_lite/internal/models"
)

func Write(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, models.ErrIncomplete):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, models.ErrNoStorage):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	default:
		if containsAny(err.Error(), "must be > 0", "part verification failed", "size mismatch", "part index out of range", "missing part") {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func containsAny(msg string, needles ...string) bool {
	for _, s := range needles {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
