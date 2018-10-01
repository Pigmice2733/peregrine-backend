package server

import (
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/events", s.eventsHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/info", s.eventHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/matches", s.matchesHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/matches/{matchKey}/info", s.matchHandler()).Methods("GET")
	return r
}
