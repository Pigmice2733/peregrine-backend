package server

import (
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// teamHandler returns a handler to get general info for a specific team
func (s *Server) teamHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		teamKey := vars["teamKey"]

		team, err := s.Store.GetTeam(r.Context(), teamKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving team info")
			return
		}

		ihttp.Respond(w, team, http.StatusOK)
	}
}

// eventTeamHandler returns a handler to get a specific team at a specific event.
func (s *Server) eventTeamHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey, teamKey := vars["eventKey"], vars["teamKey"]

		event, err := s.Store.GetEvent(r.Context(), eventKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		team, err := s.Store.GetEventTeam(r.Context(), teamKey, eventKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving team rankings data")
			return
		}

		ihttp.Respond(w, team, http.StatusOK)
	}
}

// eventTeamsHandler returns a handler to get all teams at a given event.
func (s *Server) eventTeamsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := mux.Vars(r)["eventKey"]

		event, err := s.Store.GetEvent(r.Context(), eventKey)

		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		teams, err := s.Store.GetEventTeams(r.Context(), eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving teams data")
			return
		}

		ihttp.Respond(w, teams, http.StatusOK)
	}
}
