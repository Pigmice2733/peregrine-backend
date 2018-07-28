package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func abs(a int64) int64 {
	if a < 0 {
		a *= -1
	}
	return a
}

func semiEqual(a, b, precision int64) bool {
	return abs(a-b) <= precision
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
