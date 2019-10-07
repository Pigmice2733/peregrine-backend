package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestOpenAPIHandler(t *testing.T) {
	doc := `openapi: '3.0.0'
	info:
	  version: 1.0.0
	  title: Peregrine API documentation
	servers:
	  - url: http://edge.api.peregrine.ga:8080/
	  - url: http://api.peregrine.ga:8080/
	  - url: http://localhost:8080/`

	rr := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Errorf("did not expect error %v setting up test", err)
		t.FailNow()
	}

	handler := openAPIHandler([]byte(doc))

	handler(rr, req)

	const expectedContentType = "application/x-yaml"
	actualContentType := rr.Header().Get("Content-Type")

	if actualContentType != expectedContentType {
		t.Errorf("expected content type: %s, got %s", expectedContentType, actualContentType)
	}

	actualDoc := rr.Body.String()
	if !cmp.Equal(actualDoc, doc) {
		t.Errorf("expected actual doc to match doc, but got diff: %s", cmp.Diff(doc, actualDoc))
	}
}

type mockPinger struct {
	ok bool
}

func (mp mockPinger) Ping(context.Context) error {
	if !mp.ok {
		return errors.New("could not connect")
	}

	return nil
}

func TestHealthHandler(t *testing.T) {
	testCases := []struct {
		name             string
		tbaHealthy       bool
		postgresHealthy  bool
		uptime           func() time.Duration
		expectedResponse healthStatus
	}{
		{
			name:            "all services healthy",
			tbaHealthy:      true,
			postgresHealthy: true,
			uptime:          func() time.Duration { return time.Second * 10 },
			expectedResponse: healthStatus{
				Uptime: "10s",
				Services: healthServices{
					TBA:        true,
					PostgreSQL: true,
				},
				Ok: true,
			},
		},
		{
			name:            "tba is unhealthy",
			tbaHealthy:      false,
			postgresHealthy: true,
			uptime:          func() time.Duration { return time.Second * 10 },
			expectedResponse: healthStatus{
				Uptime: "10s",
				Services: healthServices{
					TBA:        false,
					PostgreSQL: true,
				},
				Ok: false,
			},
		},
		{
			name:            "all services are unhealthy",
			tbaHealthy:      false,
			postgresHealthy: false,
			uptime:          func() time.Duration { return time.Hour * 95 },
			expectedResponse: healthStatus{
				Uptime: "95h0m0s",
				Services: healthServices{
					TBA:        false,
					PostgreSQL: false,
				},
				Ok: false,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			if err != nil {
				t.Errorf("did not expect error %v setting up request for test", err)
				t.FailNow()
			}

			handler := healthHandler(tt.uptime, mockPinger{tt.tbaHealthy}, mockPinger{tt.postgresHealthy})

			handler(rr, req)

			var actualResponse healthStatus
			if err := json.NewDecoder(rr.Body).Decode(&actualResponse); err != nil {
				t.Errorf("did not expect error %v decoding response", err)
			}

			if !cmp.Equal(actualResponse, tt.expectedResponse) {
				t.Errorf("expected actual response to match expected response, but got diff: %s", cmp.Diff(tt.expectedResponse, actualResponse))
			}
		})
	}
}
