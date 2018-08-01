package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestStatsHandler(t *testing.T) {
	weak := time.Now().Add(-time.Second * 10)
	strongUptimeGame := time.Date(2002, 3, 4, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		startTime    *time.Time
		expectedCode int
	}{
		{
			startTime:    &weak,
			expectedCode: http.StatusOK,
		},
		{
			startTime:    nil,
			expectedCode: http.StatusOK,
		},
		{
			startTime:    &strongUptimeGame,
			expectedCode: http.StatusOK,
		},
	}

	for _, testCase := range testCases {
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)

		handler := Stats(func() *time.Time { return testCase.startTime })

		handler(rr, req)

		assert.Equal(t, testCase.expectedCode, rr.Code)

		var resp struct {
			Data statsResponse `json:"data"`
		}
		assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

		expectedRunning := testCase.startTime != nil
		equalRunning := assert.Equal(t, expectedRunning, resp.Data.Running)
		if !equalRunning || (equalRunning && !resp.Data.Running) {
			return
		}

		actualStartTime, err := time.Parse(time.RFC3339, resp.Data.StartTime)
		assert.NoError(t, err)
		assert.Equal(t, testCase.startTime.Unix(), actualStartTime.Unix())

		actualNow, err := time.Parse(time.RFC3339, resp.Data.Time)
		assert.NoError(t, err)

		actualUptime, err := time.ParseDuration(resp.Data.Uptime)
		assert.NoError(t, err)

		calculatedUptime := int64(actualNow.Sub(actualStartTime).Seconds())
		assert.Equal(t, calculatedUptime, int64(actualUptime.Seconds()))
	}
}
