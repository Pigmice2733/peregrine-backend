package handlers

import (
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

// Health returns a health handler that respond with {"ok": true}
func Health() http.HandlerFunc {
	var success = map[string]bool{"ok": true}

	return func(w http.ResponseWriter, r *http.Request) {
		ihttp.Respond(w, http.StatusOK, success)
	}
}
