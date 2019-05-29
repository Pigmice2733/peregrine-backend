package server

import (
	"net/http"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"

	"github.com/fharding1/gemux"
)

func (s *Server) mux() http.Handler {
	mux := new(gemux.ServeMux)

	mux.Handle("/", http.MethodGet, healthHandler(s.uptime, s.TBA, s.Store))
	mux.Handle("/openapi.yaml", http.MethodGet, openAPIHandler(openAPI))

	mux.Handle("/authenticate", http.MethodPost, authenticateHandler(s.Logger, time.Now, s.Store, s.JWTSecret))
	mux.Handle("/refresh", http.MethodPost, refreshHandler(s.Logger, time.Now, s.Store, s.JWTSecret))

	mux.Handle("/users", http.MethodPost, s.createUserHandler())
	mux.Handle("/users", http.MethodGet, ihttp.ACL(s.getUsersHandler(), true, true, true))
	mux.Handle("/users/*", http.MethodGet, ihttp.ACL(s.getUserByIDHandler(), false, false, true))
	mux.Handle("/users/*", http.MethodPatch, ihttp.ACL(s.patchUserHandler(), false, false, true))
	mux.Handle("/users/*", http.MethodDelete, ihttp.ACL(s.deleteUserHandler(), false, false, true))

	mux.Handle("/schemas", http.MethodGet, ihttp.ACL(s.getSchemasHandler(), false, false, false))
	mux.Handle("/schemas", http.MethodPost, ihttp.ACL(s.createSchemaHandler(), true, true, true))
	mux.Handle("/schemas/*", http.MethodGet, ihttp.ACL(s.getSchemaByIDHandler(), false, false, false))

	mux.Handle("/events", http.MethodGet, s.eventsHandler())
	mux.Handle("/events/*", http.MethodPut, ihttp.ACL(s.upsertEventHandler(), true, true, true))
	mux.Handle("/events/*", http.MethodGet, s.eventHandler())

	mux.Handle("/events/*/stats", http.MethodGet, s.eventStats())

	mux.Handle("/events/*/matches", http.MethodGet, s.matchesHandler())
	mux.Handle("/events/*/matches", http.MethodPost, ihttp.ACL(s.createMatchHandler(), true, true, true))
	mux.Handle("/events/*/matches/*", http.MethodGet, s.matchHandler())

	mux.Handle("/events/*/teams", http.MethodGet, s.teamsHandler())
	mux.Handle("/events/*/teams/*", http.MethodGet, s.teamInfoHandler())
	mux.Handle("/events/*/teams/*/comments", http.MethodGet, ihttp.ACL(s.getEventComments(), false, false, false))

	mux.Handle("/events/*/matches/*/reports/*", http.MethodGet, ihttp.ACL(s.getReports(), false, false, false))
	mux.Handle("/events/*/matches/*/reports/*", http.MethodPut, ihttp.ACL(s.putReport(), false, true, true))

	mux.Handle("/events/*/matches/*/comments/*", http.MethodGet, ihttp.ACL(s.getMatchTeamComments(), false, false, false))
	mux.Handle("/events/*/matches/*/comments/*", http.MethodPut, ihttp.ACL(s.putMatchTeamComment(), false, true, true))

	mux.Handle("/leaderboard", http.MethodGet, s.leaderboardHandler())

	mux.Handle("/realms", http.MethodGet, s.realmsHandler())
	mux.Handle("/realms", http.MethodPost, ihttp.ACL(s.createRealmHandler(), true, true, true))
	mux.Handle("/realms/*", http.MethodGet, s.realmHandler())
	mux.Handle("/realms/*", http.MethodPost, ihttp.ACL(s.updateRealmHandler(), true, true, true))
	mux.Handle("/realms/*", http.MethodDelete, ihttp.ACL(s.deleteRealmHandler(), true, true, true))

	return mux
}
