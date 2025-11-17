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

// insertPart принимает PUT-запросы на запись новой части.
func (a *Server) insertPart(w http.ResponseWriter, r *http.Request) {
	req, ok := a.requirePartRequest(w, r)
	if !ok {
		return
	}
	// insertPart — thin wrapper; все проверки делаются в writePart.
	a.writePart(w, r, req)
}

// writePart создаёт директорию для файла, валидирует вход и атомарно сохраняет часть.
func (a *Server) writePart(w http.ResponseWriter, r *http.Request, req *partRequest) {
	// Для каждой части выделяем отдельную директорию по fileID, чтобы комбинировать
	// payload (*.part) и meta.json. MkdirAll безопасен при повторных вызовах.
	if err := os.MkdirAll(req.dir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Клиент может отправить ожидаемую сумму SHA-256 — сверяемся после записи.
	expSha := r.Header.Get(storageproto.HeaderChecksum)
	// Content-Length обязателен для честных стораджей, но допускаем отсутствие
	// (возвращается -1), чтобы не падать на нестандартных клиентах.
	size, err := parseContentLength(r.Header.Get("Content-Length"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Создаём временный файл для части. os.Create перетирает существующий файл,
	// что полезно для повторной догрузки той же части.
	f, err := os.Create(req.part)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Пишем тело сразу в файл и в sha256.Writer — MultiWriter избежит двойного чтения.
	h := sha256.New()
	wrt := io.MultiWriter(f, h)
	n, err := io.Copy(wrt, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Если клиент объявил Content-Length, проверяем фактически записанный объём.
	if size > 0 && n != size {
		http.Error(w, "size mismatch", http.StatusBadRequest)
		return
	}
	// Финализируем хеш и сравниваем с переданным значением (если оно было).
	got := hex.EncodeToString(h.Sum(nil))
	if expSha != "" && got != expSha {
		http.Error(w, "sha256 mismatch", http.StatusConflict)
		return
	}

	// Храним общие параметры запроса в meta.json, поэтому считываем заголовок заранее.
	totalPartsStr := r.Header.Get(storageproto.HeaderTotalParts)
	totalParts, err := strconv.Atoi(totalPartsStr)
	if err != nil || totalParts <= 0 {
		http.Error(w, "invalid total parts header", http.StatusBadRequest)
		return
	}

	// На диске ведём отдельный meta.json. writeMeta обновит/создаст его атомарно,
	// чтобы сборка файла знала о размерах и хешах сохранённых частей.
	if err = writeMeta(req.meta, req.fileID, req.idx, n, got, totalParts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Успех — возвращаем 201, как задокументировано в Storage API.
	w.WriteHeader(http.StatusCreated)
}

// parseContentLength аккуратно разбирает значение заголовка и отличает отсутствие от ошибки.
func parseContentLength(value string) (int64, error) {
	if value == "" {
		// Пустой заголовок = неизвестный размер. Возвращаем -1 и не считаем это ошибкой.
		return -1, nil
	}

	sz, err := strconv.ParseInt(value, 10, 64)
	if err != nil || sz < 0 {
		return 0, fmt.Errorf("invalid Content-Length header")
	}

	return sz, nil
}
