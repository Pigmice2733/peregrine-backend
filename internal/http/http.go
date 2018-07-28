package http

import (
	"encoding/json"
	"net/http"
)

// ResponseErrorer defines the neccessary methods for a ResponseError, the error
// message, and the error code.
type ResponseErrorer interface {
	Error() string
	Code() int
}

// ResponseError defines a response error, the error message, and error code.
type ResponseError struct {
	Message   string `json:"message"`
	ErrorCode int    `json:"code"`
}

// Error returns the ResponseError error message.
func (re ResponseError) Error() string { return re.Message }

// Code returns the ResponseError error code.
func (re ResponseError) Code() int { return re.ErrorCode }

// Response defines a JSON response, a ResponseError, if applicable, and
// data.
type Response struct {
	Error *ResponseError `json:"error,omitempty"`
	Data  interface{}    `json:"data,omitempty"`
}

// Respond JSON encodes the given value and writes it to the ResponseWriter,
// and sets the given http code.
func Respond(w http.ResponseWriter, httpCode int, args ...interface{}) {
	w.WriteHeader(httpCode)
	w.Header().Set("Content-Type", "application/json")

	var resp Response
	for _, v := range args {
		switch v := v.(type) {
		case ResponseErrorer:
			resp.Error = &ResponseError{v.Error(), v.Code()}
		default:
			resp.Data = v
		}
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		Error(w, http.StatusInternalServerError)
	}
}

// Error writes an HTTP error code.
func Error(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
