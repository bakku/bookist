package web

import (
	"net/http"

	"bakku.dev/bookist/internal/covers"
)

func (s *Server) handleBookCover(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	file, err := s.covers.Open(key)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	defer func() {
		_ = file.Close()
	}()

	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", covers.MediaType(key))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	http.ServeContent(w, r, key, info.ModTime(), file)
}
