package server

import (
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()

	r.Handle("/events", s.eventsHandler()).Methods("GET")

	return r
}
