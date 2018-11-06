package server

import (
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()

	r.Handle("/", s.healthHandler()).Methods("GET")

	r.Handle("/authenticate", s.authenticateHandler()).Methods("POST")
	r.Handle("/users", ihttp.ACL(s.createUserHandler(), true, false)).Methods("POST")
	r.Handle("/users", ihttp.ACL(s.getUsersHandler(), true, true)).Methods("GET")
	r.Handle("/users/{id}", s.getUserByIDHandler()).Methods("GET")
	r.Handle("/users/{id}", s.patchUserHandler()).Methods("PATCH")

	r.Handle("/events", s.eventsHandler()).Methods("GET")
	r.Handle("/events", ihttp.ACL(s.createEventHandler(), false, true)).Methods("PUT")
	r.Handle("/events/{eventKey}", s.eventHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/matches", s.matchesHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/matches", ihttp.ACL(s.createMatchHandler(), false, true)).Methods("PUT")
	r.Handle("/events/{eventKey}/matches/{matchKey}", s.matchHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/teams", s.teamsHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/teams/{teamKey}", s.teamInfoHandler()).Methods("GET")

	return r
}
