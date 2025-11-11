package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sir_venger/s3_lite/internal/app/resthttp"
	"github.com/sir_venger/s3_lite/internal/app/storagehttp"
	"github.com/sir_venger/s3_lite/internal/config"
)

func Test_UploadDownload_DistinctAndIntegrity(t *testing.T) {
	// 3 in-memory storage nodes
	s1 := httptest.NewServer(storagehttp.New(t.TempDir()))
	s2 := httptest.NewServer(storagehttp.New(t.TempDir()))
	s3 := httptest.NewServer(storagehttp.New(t.TempDir()))
	t.Cleanup(func() { s1.Close(); s2.Close(); s3.Close() })

	cfg := &config.Config{ListenAddr: ":0", MetaDSN: "postgres://storage:storage@localhost:5432/storage_lite?sslmode=disable", Storages: []string{s1.URL, s2.URL, s3.URL}}
	h, _, err := resthttp.NewServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rest := httptest.NewServer(h)
	t.Cleanup(rest.Close)

	payload := bytes.Repeat([]byte{0xA1, 0xB2, 0xC3, 0xD4}, 1<<18) // ~1MB
	want := sha256.Sum256(payload)

	resp, err := http.Post(rest.URL+"/files", "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		t.Fatalf("upload status %s", resp.Status)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	fileID := jsonGet(b, "file_id")
	if fileID == "" {
		t.Fatalf("no file_id in %q", string(b))
	}

	resp, err = http.Get(rest.URL + "/files/" + fileID)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	gh := sha256.Sum256(got)
	if hex.EncodeToString(gh[:]) != hex.EncodeToString(want[:]) {
		t.Fatalf("sha mismatch")
	}
}

func jsonGet(b []byte, key string) string {
	// мини-парсер json: {"key":"value"}
	k := []byte("\"" + key + "\":\"")
	i := bytes.Index(b, k)
	if i < 0 {
		return ""
	}
	j := i + len(k)
	for j < len(b) && b[j] != '"' {
		j++
	}
	return string(b[i+len(k) : j])
}
