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
var key = "196a218517b32b00959932a6e1c938cf.png"

// ── NewStore ──────────────────────────────────────────────────────────────────

func TestNewStoreCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "book-covers")

	if _, err := covers.NewStore(dir); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("expected cover directory, got info=%v err=%v", info, err)
	}
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestStoreSaveWritesImageWithOpaqueKey(t *testing.T) {
	dir := t.TempDir()

	store, err := covers.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	key, err := store.Save(pngImage)
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Ext(key) != ".png" {
		t.Fatalf("expected an opaque PNG key, got %q", key)
	}

	got, err := os.ReadFile(filepath.Join(dir, key))
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

// ── Open ──────────────────────────────────────────────────────────────────────

func TestStoreOpenReadsManagedImage(t *testing.T) {
	dir := t.TempDir()

	store, err := covers.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(dir, key), pngImage, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	file, err := store.Open(key)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = file.Close()
	}()

	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got, pngImage) {
		t.Fatalf("expected opened bytes %v, got %v", pngImage, got)
	}
}

func TestStoreOpenRejectsUnmanagedKey(t *testing.T) {
	store, err := covers.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Open("../secret.png"); err == nil {
		t.Fatalf("expected error for traversal key, got no error")
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestStoreDeleteRemovesManagedImage(t *testing.T) {
	dir := t.TempDir()
	store, err := covers.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	key, err := store.Save(pngImage)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, key)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected deleted cover to be absent, got %v", err)
	}
}

func TestStoreDeleteRejectsUnmanagedKey(t *testing.T) {
	store, err := covers.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("../secret.png"); err == nil {
		t.Fatal("expected invalid delete key to fail")
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
