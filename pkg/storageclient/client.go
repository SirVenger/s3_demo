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
	body := req.Reader
	var bar *progressBar
	if body != nil {
		bar = newProgressBar(
			fmt.Sprintf("Uploading %s part %d/%d", req.FileID, req.Index+1, req.TotalParts),
			req.Size,
		)
		body = io.TeeReader(req.Reader, progressWriter{bar: bar})
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, u, body)
	if err != nil {
		if bar != nil {
			bar.Fail(err)
		}
		return err
	}
	if bar != nil {
		bar.render(true, "")
	}

	httpReq.Header.Set("Content-Length", strconv.FormatInt(req.Size, 10))
	if req.Sha256 != "" {
		httpReq.Header.Set(storageproto.HeaderChecksum, req.Sha256)
	}
	httpReq.Header.Set(storageproto.HeaderTotalParts, strconv.Itoa(req.TotalParts))

	resp, err := h.c.Do(httpReq)
	if err != nil {
		if bar != nil {
			bar.Fail(err)
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		err = fmt.Errorf("storage PUT failed: %s", resp.Status)
		if bar != nil {
			bar.Fail(err)
		}
		return err
	}

	if bar != nil {
		bar.Finish()
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
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("storage GET failed: %s", resp.Status)
	}

	expectedSize := resp.ContentLength
	if expectedSize <= 0 {
		if header := resp.Header.Get(storageproto.HeaderPartSize); header != "" {
			if sz, parseErr := strconv.ParseInt(header, 10, 64); parseErr == nil && sz > 0 {
				expectedSize = sz
			}
		}
	}

	bar := newProgressBar(
		fmt.Sprintf("Downloading %s part %d", fileID, index),
		expectedSize,
	)
	bar.render(true, "")

	return newProgressReadCloser(resp.Body, bar), nil
}
