package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	handler := Health()

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)

	handler(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)

	var body ihttp.Response
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&body))

	assert.Nil(t, body.Error)

	data, ok := body.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, data["ok"])
}
