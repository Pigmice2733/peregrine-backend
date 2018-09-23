package server

import (
	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/events", gziphandler.GzipHandler(s.eventsHandler())).Methods("GET")
	return r
}
