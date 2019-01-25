package http

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

// Error sets the specified HTTP status code.
func Error(w http.ResponseWriter, httpCode int) {
	http.Error(w, http.StatusText(httpCode), httpCode)
}

// Respond encodes the data and ResponseError to JSON and responds with it and
// the http code. If the encoding fails, sets an InternalServerError.
func Respond(w http.ResponseWriter, data interface{}, httpCode int) {
	var resp interface{}
	if v, ok := data.(error); ok {
		resp = errorResponse{Error: v.Error()}
	} else {
		resp = data
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)

	if resp != nil {
		json.NewEncoder(w).Encode(resp)
	}
}
