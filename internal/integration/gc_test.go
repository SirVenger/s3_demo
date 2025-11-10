package integration

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	storagehttp "github.com/yourname/storage_lite/internal/app/storagehttp"
)

func Test_StorageGC_RemovesStaleDirs(t *testing.T) {
	root := t.TempDir()
	h := storagehttp.New(root)
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)

	// создаём полу-загрузку: meta.json с total_parts=6, parts < 6
	d := filepath.Join(root, "file123")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(d, "meta.json"), []byte(`{"file_id":"file123","total_parts":6,"parts":{"0":{"index":0,"size":10,"sha256":"x"}}}`), 0o644)
	// старим модтайм
	old := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(filepath.Join(d, "meta.json"), old, old)

	// однократный запуск sweep (ttl 24h)
	if err := storagehttpTestSweepOnce(root, 24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(d); !os.IsNotExist(err) {
		t.Fatalf("stale dir not removed")
	}
}

// storagehttp.sweepOnce неэкспортируемая, сделаем тонкий враппер в тесте
func storagehttpTestSweepOnce(root string, ttl time.Duration) error { return callSweepOnce(root, ttl) }
