package server

import (
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/handlers"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

// initRoutesV1 creates the server router and sets all v1 routes.
func (s *Server) initRoutesV1() {
	r := mux.NewRouter()
	r.Use(ihttp.CORS)

	r.HandleFunc("/", handlers.Stats(func() *time.Time {
		return s.startTime
	}))

	routerV1 := r.PathPrefix("/v1").Subrouter()

	routerV1.HandleFunc("/health", handlers.Health()).Methods("GET")

	s.router = r
}
