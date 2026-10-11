package api

import (
	"net/http"
	"strconv"

	"bakku.dev/bookist/internal/apperr"
)

// parsePathID reads a positive ID from the named path value. On failure it
// writes a 400 naming the resource, for example "invalid book ID".
func parsePathID(w http.ResponseWriter, r *http.Request, name, label string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, apperr.Invalid("invalid_id", "invalid "+label+" ID", name))
		return 0, false
	}
	return id, true
}
