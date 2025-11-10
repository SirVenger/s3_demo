package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/yourname/storage_lite/internal/app/resthttp"
	"github.com/yourname/storage_lite/internal/app/storagehttp"
	"github.com/yourname/storage_lite/internal/config"
)

type uploadResponse struct {
	FileID string `json:"file_id"`
	Size   int64  `json:"size"`
	Parts  int    `json:"parts"`
}

func TestUploadAndStreamFile(t *testing.T) {
	storageDir1 := t.TempDir()
	storageDir2 := t.TempDir()

	storageSrv1 := httptest.NewServer(storagehttp.New(storageDir1))
	t.Cleanup(storageSrv1.Close)

	storageSrv2 := httptest.NewServer(storagehttp.New(storageDir2))
	t.Cleanup(storageSrv2.Close)

	metaPath := filepath.Join(t.TempDir(), "meta.db")
	cfg := &config.Config{
		ListenAddr: ":0",
		MetaPath:   metaPath,
		Storages:   []string{storageSrv1.URL},
	}

	handler, _, err := resthttp.NewServer(cfg)
	if err != nil {
		t.Fatalf("new rest server: %v", err)
	}
	restSrv := httptest.NewServer(handler)
	t.Cleanup(restSrv.Close)

	addReq := map[string][]string{"storages": {storageSrv2.URL}}
	if err := postJSON(restSrv.URL+"/admin/storages", addReq); err != nil {
		t.Fatalf("add storages: %v", err)
	}

	payload := bytes.Repeat([]byte("0123456789abcdef"), 1024) // 16 KiB
	uploadRes, err := uploadFile(restSrv.URL+"/files", payload)
	if err != nil {
		t.Fatalf("upload file: %v", err)
	}
	if uploadRes.FileID == "" {
		t.Fatalf("empty file id in response")
	}

	got, err := downloadFile(restSrv.URL + "/files/" + uploadRes.FileID)
	if err != nil {
		t.Fatalf("download file: %v", err)
	}

	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded data mismatch, got %d bytes want %d", len(got), len(payload))
	}
}

func uploadFile(url string, data []byte) (uploadResponse, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return uploadResponse{}, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return uploadResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return uploadResponse{}, fmt.Errorf("unexpected status %s: %s", resp.Status, string(body))
	}

	var out uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return uploadResponse{}, err
	}
	return out, nil
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %s: %s", resp.Status, string(body))
	}
	return io.ReadAll(resp.Body)
}

func postJSON(url string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %s: %s", resp.Status, string(body))
	}
	return nil
}
