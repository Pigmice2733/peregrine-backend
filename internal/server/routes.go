package server

import (
	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/events", gziphandler.GzipHandler(s.eventsHandler())).Methods("GET")
	r.Handle("/events/{eventKey}/info", gziphandler.GzipHandler(s.eventHandler())).Methods("GET")
	r.Handle("/events/{eventKey}/matches", gziphandler.GzipHandler(s.matchesHandler())).Methods("GET")
	r.Handle("/events/{eventKey}/matches/{matchKey}/info", gziphandler.GzipHandler(s.matchHandler())).Methods("GET")
	return r
}
