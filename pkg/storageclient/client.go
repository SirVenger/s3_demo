package storageclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/sir_venger/s3_lite/pkg/storageproto"
)

type PutPartRequest struct {
	FileID     string
	Index      int
	Reader     io.Reader
	Size       int64
	Sha256     string
	TotalParts int
}

type Client interface {
	// PutPart Положить часть файла в хранилище
	PutPart(ctx context.Context, baseURL string, req PutPartRequest) error
	// GetPart Достать часть файла в хранилище
	GetPart(ctx context.Context, baseURL, fileID string, index int) (io.ReadCloser, error)
}

type httpClient struct {
	c *http.Client
}

// New создаёт HTTP-клиент по умолчанию.
func New() Client {
	return &httpClient{
		c: &http.Client{},
	}
}

// PutPart загружает часть файла в указанный storage.
func (h *httpClient) PutPart(ctx context.Context, baseURL string, req PutPartRequest) error {
	u := fmt.Sprintf(storageproto.PartsPathFormat, baseURL, req.FileID, req.Index)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, u, req.Reader)
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Length", strconv.FormatInt(req.Size, 10))
	if req.Sha256 != "" {
		httpReq.Header.Set(storageproto.HeaderChecksum, req.Sha256)
	}
	httpReq.Header.Set(storageproto.HeaderTotalParts, strconv.Itoa(req.TotalParts))

	resp, err := h.c.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("storage PUT failed: %s", resp.Status)
	}

	return nil
}

// GetPart скачивает часть файла и возвращает поток с телом.
func (h *httpClient) GetPart(ctx context.Context, baseURL, fileID string, index int) (io.ReadCloser, error) {
	u := fmt.Sprintf(storageproto.PartsPathFormat, baseURL, fileID, index)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("storage GET failed: %s", resp.Status)
	}

	return resp.Body, nil
}
