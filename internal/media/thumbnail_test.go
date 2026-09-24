package media

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func countCache(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(os.TempDir(), "vidctl-thumbs"))
	if err != nil {
		t.Fatalf("falha ao ler o diretório de cache: %v", err)
	}
	return len(entries)
}

func TestThumbDataURL(t *testing.T) {
	input := makeTestVideo(t)

	url, err := ThumbDataURL(input, 2)
	if err != nil {
		t.Fatalf("ThumbDataURL: %v", err)
	}
	if !strings.HasPrefix(url, "data:image/jpeg;base64,") {
		t.Errorf("data-URL inesperada: %.40s…", url)
	}
}

func TestThumbDataURLCache(t *testing.T) {
	input := makeTestVideo(t)

	if _, err := ThumbDataURL(input, 2); err != nil {
		t.Fatalf("ThumbDataURL: %v", err)
	}
	before := countCache(t)

	if _, err := ThumbDataURL(input, 2); err != nil {
		t.Fatalf("ThumbDataURL (cache): %v", err)
	}
	if got := countCache(t); got != before {
		t.Errorf("cache hit gerou arquivo novo: %d != %d", got, before)
	}

	now := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(input, now, now); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	if _, err := ThumbDataURL(input, 2); err != nil {
		t.Fatalf("ThumbDataURL (mtime novo): %v", err)
	}
	if got := countCache(t); got != before+1 {
		t.Errorf("mudança de mtime deveria invalidar o cache: %d != %d", got, before+1)
	}
}

func TestThumbDataURLMissingFile(t *testing.T) {
	if _, err := ThumbDataURL(filepath.Join(t.TempDir(), "nope.mp4"), 2); err == nil {
		t.Fatal("esperava erro para arquivo inexistente")
	}
}
