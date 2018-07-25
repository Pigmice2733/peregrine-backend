package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type usernameError struct {
	username string
}

func (ue usernameError) Error() string {
	return fmt.Sprintf("Username '%s' is either too long or too short.", ue.username)
}

func (usernameError) Code() int { return 0 }

var frankUsernameError = usernameError{"frank"}

func TestRespond(t *testing.T) {
	var testCases = []struct {
		httpCode             int
		args                 []interface{}
		expectedErrorMessage string
		expectedErrorCode    int
		expectedData         interface{}
	}{
		{httpCode: http.StatusOK, args: []interface{}{true}, expectedData: true},
		{
			httpCode:             http.StatusNotFound,
			args:                 []interface{}{frankUsernameError},
			expectedErrorMessage: frankUsernameError.Error(),
			expectedErrorCode:    frankUsernameError.Code(),
			expectedData:         nil,
		},
		{
			httpCode:             http.StatusNotFound,
			args:                 []interface{}{42, frankUsernameError},
			expectedErrorMessage: frankUsernameError.Error(),
			expectedErrorCode:    frankUsernameError.Code(),
			expectedData:         float64(42),
		},
		{
			httpCode:             http.StatusNotFound,
			args:                 []interface{}{42, ResponseError{Message: "bla", ErrorCode: 4}, "hello, world"},
			expectedErrorMessage: "bla",
			expectedErrorCode:    4,
			expectedData:         "hello, world",
		},
	}

	for _, testCase := range testCases {
		rr := httptest.NewRecorder()

		Respond(rr, testCase.httpCode, testCase.args...)

		assert.Equal(t, testCase.httpCode, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var body Response
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&body))

		assert.Equal(t, testCase.expectedData, body.Data)

		if assert.Equal(t, testCase.expectedErrorMessage != "", body.Error != nil) && body.Error != nil {
			assert.Equal(t, testCase.expectedErrorMessage, body.Error.Error())
			assert.Equal(t, testCase.expectedErrorCode, body.Error.Code())
		}
	}
}

func TestError(t *testing.T) {
	rr := httptest.NewRecorder()

	Error(rr, http.StatusOK)
	http.StatusText(http.StatusOK)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, http.StatusText(http.StatusOK), strings.Trim(rr.Body.String(), "\n"))
}
