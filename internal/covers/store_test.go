package covers_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"bakku.dev/bookist/internal/covers"
)

var pngImage = []byte("\x89PNG\r\n\x1a\ncover")

// ── Save ──────────────────────────────────────────────────────────────────────

func TestStoreCreatesDirectoryAndSavesImage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "book-covers")
	store, err := covers.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	key, err := store.Save(pngImage)
	if err != nil {
		t.Fatal(err)
	}
	if !covers.ValidKey(key) || filepath.Ext(key) != ".png" {
		t.Fatalf("expected an opaque PNG key, got %q", key)
	}

	file, err := store.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, pngImage) {
		t.Fatalf("expected saved bytes %v, got %v", pngImage, got)
	}
}

func TestStoreRejectsUnsupportedAndOversizedImages(t *testing.T) {
	store, err := covers.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Save([]byte("not an image")); !errors.Is(err, covers.ErrUnsupportedMediaType) {
		t.Fatalf("expected ErrUnsupportedMediaType, got %v", err)
	}
	tooLarge := make([]byte, covers.MaxSize+1)
	copy(tooLarge, pngImage)
	if _, err := store.Save(tooLarge); !errors.Is(err, covers.ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}

// ── Validation ────────────────────────────────────────────────────────────────

func TestValidateAcceptsSupportedFormats(t *testing.T) {
	for name, image := range map[string][]byte{
		"JPEG": {0xff, 0xd8, 0xff, 0xe0},
		"PNG":  pngImage,
		"WebP": []byte("RIFF\x00\x00\x00\x00WEBPVP"),
	} {
		t.Run(name, func(t *testing.T) {
			if err := covers.Validate(image); err != nil {
				t.Fatalf("expected %s to be accepted: %v", name, err)
			}
		})
	}
}

// ── Managed Keys ──────────────────────────────────────────────────────────────

func TestStoreRejectsUnmanagedKeys(t *testing.T) {
	store, err := covers.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Open("../secret.png"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not found for traversal key, got %v", err)
	}
	if err := store.Delete("../secret.png"); err == nil {
		t.Fatal("expected invalid delete key to fail")
	}
}
