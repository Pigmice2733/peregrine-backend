package server

import (
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/events", s.eventsHandler()).Methods("GET")
	return r
}
