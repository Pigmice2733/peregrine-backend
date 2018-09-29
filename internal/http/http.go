package http

import (
	"encoding/json"
	"net/http"
)

// ResponseError defines a response error, the error message, and error code.
type ResponseError struct {
	Message   string `json:"message"`
	ErrorCode int    `json:"code"`
}

// Response is the correct structure for scouting API responses.
type Response struct {
	Error *ResponseError `json:"error,omitempty"`
	Data  interface{}    `json:"data,omitempty"`
}

// Error sets the specified HTTP status code.
func Error(w http.ResponseWriter, httpCode int) {
	http.Error(w, http.StatusText(httpCode), httpCode)
}

// Respond encodes the data and ResponseError to JSON and responds with it and
// the http code. If the encoding fails, sets an InternalServerError.
func Respond(w http.ResponseWriter, data interface{}, respErr *ResponseError, httpCode int) {
	response := Response{
		Error: respErr,
		Data:  data,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		Error(w, http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	_, _ = w.Write(jsonData) // make linters happy
}
