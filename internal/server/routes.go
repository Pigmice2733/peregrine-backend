package server

import (
	"github.com/Pigmice2733/peregrine-backend/internal/handlers"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

// initRoutesV1 creates the server router and sets all v1 routes.
func (s *Server) initRoutesV1() {
	r := mux.NewRouter().PathPrefix("/v1").Subrouter()
	r.Use(ihttp.CORS, ihttp.JSON)

	r.HandleFunc("/health", handlers.Health()).Methods("GET")

	s.router = r
}
