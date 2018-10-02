package server

import (
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()

	r.Handle("/authenticate", s.authenticateHandler()).Methods("POST")
	r.Handle("/users", s.authMiddleware(s.createUserHandler(), true, false)).Methods("POST")

	r.Handle("/events", s.eventsHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/info", s.eventHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/matches", s.matchesHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/matches/{matchKey}/info", s.matchHandler()).Methods("GET")

	return r
}
