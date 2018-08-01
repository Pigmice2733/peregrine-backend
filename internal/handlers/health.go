package handlers

import (
	"net/http"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

// Health returns a health handler that respond with {"ok": true}
func Health() http.HandlerFunc {
	var success = map[string]bool{"ok": true}

	return func(w http.ResponseWriter, r *http.Request) {
		ihttp.Respond(w, http.StatusOK, success)
	}
}

type getStartTimeFunc func() *time.Time

type statsResponse struct {
	Running   bool   `json:"running"`
	StartTime string `json:"startTime"`
	Time      string `json:"time"`
	Uptime    string `json:"uptime"`
}

// Stats returns a handler that returns server start time and uptime.
func Stats(getServerStartTime getStartTimeFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := getServerStartTime()
		now := time.Now()

		resp := statsResponse{Running: startTime != nil, Time: now.Format(time.RFC3339)}
		if startTime != nil {
			resp.StartTime = startTime.Format(time.RFC3339)
			resp.Uptime = now.Sub(*startTime).Truncate(time.Second).String()
		}

		ihttp.Respond(w, http.StatusOK, resp)
	}
}
