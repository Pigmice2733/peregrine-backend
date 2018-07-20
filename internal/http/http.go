package http

import (
	"encoding/json"
	"net/http"
)

// Respond JSON encodes the given value and writes it to the ResponseWriter,
// and sets the given http code.
func Respond(w http.ResponseWriter, code int, v interface{}) {
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		Error(w, http.StatusInternalServerError)
	}
}

// Error writes an HTTP error code.
func Error(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
