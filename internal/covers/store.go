package covers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

const MaxSize = 10 << 20

var ErrTooLarge = errors.New("cover must not exceed 10 MiB")
var ErrUnsupportedMediaType = errors.New("cover must be a JPEG, PNG, or WebP image")

var keyPattern = regexp.MustCompile(`^[0-9a-f]{32}\.(jpg|png|webp)$`)

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cover directory: %w", err)
	}

	return &Store{dir: dir}, nil
}

func (s *Store) Save(data []byte) (string, error) {
	extension, err := validate(data)
	if err != nil {
		return "", err
	}

	for {
		key, err := randomKey(extension)
		if err != nil {
			return "", err
		}

		file, err := os.OpenFile(filepath.Join(s.dir, key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("create cover: %w", err)
		}

		if _, err := file.Write(data); err != nil {
			_ = file.Close()
			_ = os.Remove(filepath.Join(s.dir, key))
			return "", fmt.Errorf("write cover: %w", err)
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(filepath.Join(s.dir, key))
			return "", fmt.Errorf("close cover: %w", err)
		}

		return key, nil
	}
}

func Validate(data []byte) error {
	_, err := validate(data)
	return err
}

func (s *Store) Delete(key string) error {
	if !ValidKey(key) {
		return fmt.Errorf("invalid cover key")
	}
	if err := os.Remove(filepath.Join(s.dir, key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete cover: %w", err)
	}
	return nil
}

func (s *Store) Open(key string) (*os.File, error) {
	if !ValidKey(key) {
		return nil, os.ErrNotExist
	}
	return os.Open(filepath.Join(s.dir, key))
}

func ValidKey(key string) bool {
	return keyPattern.MatchString(key)
}

func MediaType(key string) string {
	switch filepath.Ext(key) {
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func extensionFor(data []byte) (string, error) {
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}

	switch http.DetectContentType(sample) {
	case "image/jpeg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	case "image/webp":
		return ".webp", nil
	default:
		return "", ErrUnsupportedMediaType
	}
}

func validate(data []byte) (string, error) {
	if len(data) > MaxSize {
		return "", ErrTooLarge
	}

	return extensionFor(data)
}

func randomKey(extension string) (string, error) {
	random := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return "", fmt.Errorf("generate cover key: %w", err)
	}
	return hex.EncodeToString(random) + extension, nil
}
