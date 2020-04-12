package server

import (
	"net/http"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

func (s *Server) registerRoutes() *mux.Router {
	r := mux.NewRouter()

	r.Handle("/", healthHandler(s.uptime, s.TBA, s.Store)).Methods(http.MethodGet)
	r.Handle("/openapi.yaml", openAPIHandler(openAPI)).Methods(http.MethodGet)

	r.Handle("/authenticate", authenticateHandler(s.Logger, time.Now, s.Store, s.JWTSecret)).Methods(http.MethodPost)
	r.Handle("/refresh", refreshHandler(s.Logger, time.Now, s.Store, s.JWTSecret)).Methods(http.MethodPost)

	r.Handle("/users", s.createUserHandler()).Methods(http.MethodPost)
	r.Handle("/users", ihttp.ACL(s.getUsersHandler(), false, false, true)).Methods(http.MethodGet)
	r.Handle("/users/{id}", ihttp.ACL(s.getUserByIDHandler(), false, false, true)).Methods(http.MethodGet)
	r.Handle("/users/{id}", ihttp.ACL(s.patchUserHandler(), false, false, true)).Methods(http.MethodPatch)
	r.Handle("/users/{id}", ihttp.ACL(s.deleteUserHandler(), false, false, true)).Methods(http.MethodDelete)

	r.Handle("/schemas", ihttp.ACL(s.getSchemasHandler(), false, false, false)).Methods(http.MethodGet)
	r.Handle("/schemas", ihttp.ACL(s.createSchemaHandler(), true, true, true)).Methods(http.MethodPost)
	r.Handle("/schemas/{id}", ihttp.ACL(s.getSchemaByIDHandler(), false, false, false)).Methods(http.MethodGet)

	r.Handle("/years", s.eventYearsHandler()).Methods(http.MethodGet)

	r.Handle("/events", s.eventsHandler()).Methods(http.MethodGet)
	r.Handle("/events/{eventKey}", ihttp.ACL(s.upsertEventHandler(), true, true, true)).Methods(http.MethodPut)
	r.Handle("/events/{eventKey}", s.eventHandler()).Methods(http.MethodGet)

	r.Handle("/events/{eventKey}/stats", s.eventStats()).Methods(http.MethodGet)

	r.Handle("/events/{eventKey}/matches", s.matchesHandler()).Methods(http.MethodGet)
	r.Handle("/events/{eventKey}/matches/{matchKey}", s.matchHandler()).Methods(http.MethodGet)
	r.Handle("/events/{eventKey}/matches/{matchKey}", ihttp.ACL(s.upsertMatchHandler(), true, true, true)).Methods(http.MethodPut)
	r.Handle("/events/{eventKey}/matches/{matchKey}", ihttp.ACL(s.deleteMatchHandler(), true, true, true)).Methods(http.MethodDelete)

	r.Handle("/events/{eventKey}/teams", s.eventTeamsHandler()).Methods(http.MethodGet)
	r.Handle("/events/{eventKey}/teams/{teamKey}", s.eventTeamHandler()).Methods(http.MethodGet)

	r.Handle("/events/{eventKey}/matches/{matchKey}/teams/{teamKey}/stats", s.matchTeamStats()).Methods(http.MethodGet)

	r.Handle("/reports", ihttp.ACL(s.postReportHandler(), false, true, true)).Methods(http.MethodPost)
	r.Handle("/reports", ihttp.ACL(s.getReportsHandler(), false, false, false)).Methods(http.MethodGet)
	r.Handle("/reports/{id}", ihttp.ACL(s.putReportHandler(), false, true, true)).Methods(http.MethodPut)
	r.Handle("/reports/{id}", ihttp.ACL(s.deleteReportHandler(), false, true, true)).Methods(http.MethodDelete)

	r.Handle("/leaderboard", s.leaderboardHandler()).Methods(http.MethodGet)

	r.Handle("/realms", s.realmsHandler()).Methods(http.MethodGet)
	r.Handle("/realms", s.createRealmHandler()).Methods(http.MethodPost)
	r.Handle("/realms/{id}", s.realmHandler()).Methods(http.MethodGet)
	r.Handle("/realms/{id}", ihttp.ACL(s.updateRealmHandler(), true, true, true)).Methods(http.MethodPost)
	r.Handle("/realms/{id}", ihttp.ACL(s.deleteRealmHandler(), true, true, true)).Methods(http.MethodDelete)

	r.Handle("/teams/{teamKey}", s.teamHandler()).Methods(http.MethodGet)

	return r
}
