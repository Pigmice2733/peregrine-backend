package http

import (
	"encoding/json"
	"net/http"

	"gopkg.in/go-playground/validator.v9"
)

type response struct {
	Data interface{} `json:"data,omitempty"`
	Err  interface{} `json:"error,omitempty"`
}

// Error sets the specified HTTP status code.
func Error(w http.ResponseWriter, httpCode int) {
	http.Error(w, http.StatusText(httpCode), httpCode)
}

// Respond encodes the data and ResponseError to JSON and responds with it and
// the http code. If the encoding fails, sets an InternalServerError.
func Respond(w http.ResponseWriter, data interface{}, httpCode int) {
	var resp response
	switch v := data.(type) {
	case validator.ValidationErrors:
		resp.Err = v.Error()
	case error:
		resp.Err = v.Error()
	default:
		resp.Data = v
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		Error(w, http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	_, _ = w.Write(jsonData) // make linters happy
}
