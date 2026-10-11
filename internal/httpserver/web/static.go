package web

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"path"
	"time"
)

// staticHandler serves embedded static files with a content-hash ETag.
// Embedded files have no modification time, so without it browsers could never
// revalidate and would download every asset again on each page load.
type staticHandler struct {
	files fs.FS
	etags map[string]string
}

func newStaticHandler(files fs.FS) (*staticHandler, error) {
	etags := map[string]string{}

	err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		file, err := files.Open(name)
		if err != nil {
			return err
		}
		defer func() {
			_ = file.Close()
		}()

		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			return err
		}

		etags[name] = `"` + hex.EncodeToString(hash.Sum(nil)) + `"`
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &staticHandler{files: files, etags: etags}, nil
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := path.Clean(r.URL.Path)
	etag, ok := h.etags[name]
	if !ok {
		http.NotFound(w, r)
		return
	}

	file, err := h.files.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	content, ok := file.(io.ReadSeeker)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")

	// ServeContent answers a matching If-None-Match with 304 and handles ranges.
	http.ServeContent(w, r, name, time.Time{}, content)
}
