package server

import (
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()

	r.Handle("/", s.healthHandler()).Methods("GET")

	r.Handle("/authenticate", s.authenticateHandler()).Methods("POST")
	r.Handle("/users", s.createUserHandler()).Methods("POST")
	r.Handle("/users", ihttp.ACL(s.getUsersHandler(), true, true, true)).Methods("GET")
	r.Handle("/users/{id}", ihttp.ACL(s.getUserByIDHandler(), false, false, true)).Methods("GET")
	r.Handle("/users/{id}", ihttp.ACL(s.patchUserHandler(), false, false, true)).Methods("PATCH")
	r.Handle("/users/{id}", ihttp.ACL(s.deleteUserHandler(), false, false, true)).Methods("DELETE")

	r.Handle("/events", s.eventsHandler()).Methods("GET")
	r.Handle("/events", ihttp.ACL(s.createEventHandler(), false, true, true)).Methods("POST")
	r.Handle("/events/{eventKey}", s.eventHandler()).Methods("GET")

	r.Handle("/events/{eventKey}/matches", s.matchesHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/matches", ihttp.ACL(s.createMatchHandler(), false, true, true)).Methods("POST")
	r.Handle("/events/{eventKey}/matches/{matchKey}", s.matchHandler()).Methods("GET")

	r.Handle("/events/{eventKey}/teams", s.teamsHandler()).Methods("GET")
	r.Handle("/events/{eventKey}/teams/{teamKey}", s.teamInfoHandler()).Methods("GET")

	r.Handle("/realms", ihttp.ACL(s.realmsHandler(), false, true, true)).Methods("GET")
	r.Handle("/realms", ihttp.ACL(s.createRealmHandler(), false, true, true)).Methods("POST")
	r.Handle("/realms/{teamKey}", ihttp.ACL(s.realmHandler(), false, true, true)).Methods("GET")
	r.Handle("/realms/{teamKey}", ihttp.ACL(s.patchRealmHandler(), false, true, true)).Methods("PATCH")
	r.Handle("/realms/{teamKey}", ihttp.ACL(s.deleteRealmHandler(), false, true, true)).Methods("DELETE")

	return r
}
