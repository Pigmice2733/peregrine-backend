package server

import (
	"context"
	"net/http"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// teamHandler returns a handler to get general info for a specific team
func (s *Server) teamHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.updateTeams(r.Context()); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("unable to update teams data")
			return
		}

		vars := mux.Vars(r)
		teamKey := vars["teamKey"]

		team, err := s.Store.GetTeam(r.Context(), teamKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving team info")
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
			go s.Logger.WithError(err).Error("retrieving event")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		// Get new team rankings data from TBA
		err = s.updateEventTeamRankings(r.Context(), eventKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("updating team rankings data")
			return
		}

		team, err := s.Store.GetEventTeam(r.Context(), teamKey, eventKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving team rankings data")
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
			go s.Logger.WithError(err).Error("retrieving event")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		// Get new team rankings data from TBA
		err = s.updateEventTeamRankings(r.Context(), eventKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("updating team rankings data")
			return
		}

		teams, err := s.Store.GetEventTeams(r.Context(), eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving teams data")
			return
		}

		ihttp.Respond(w, teams, http.StatusOK)
	}
}

// Get new team key data from TBA for a particular event. Upsert data into database.
func (s *Server) updateEventTeamKeys(ctx context.Context, eventKey string) error {
	// Check that eventKey is a valid event key
	valid, err := s.Store.CheckTBAEventKeyExists(ctx, eventKey)
	if err != nil {
		return err
	}
	if !valid {
		return nil
	}

	teams, err := s.TBA.GetTeamKeys(ctx, eventKey)
	if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
		return nil
	} else if err != nil {
		return err
	}

	return s.Store.EventTeamKeysUpsert(ctx, eventKey, teams)
}

// Get new team rankings data from TBA for a particular event. Upsert data into database.
func (s *Server) updateEventTeamRankings(ctx context.Context, eventKey string) error {
	// Check that eventKey is a valid event key
	valid, err := s.Store.CheckTBAEventKeyExists(ctx, eventKey)
	if err != nil {
		return err
	}
	if !valid {
		return nil
	}

	teams, err := s.TBA.GetTeamRankings(ctx, eventKey)
	if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
		return nil
	} else if err != nil {
		return err
	}

	return s.Store.EventTeamsUpsert(ctx, teams)
}

// Hours to cache teams data from TBA for
const teamsExpiry = 6.0

// Get new teams data from TBA only if data are over 3 hours old.
// Upsert teams data into database.
func (s *Server) updateTeams(ctx context.Context) error {
	now := time.Now()

	if s.teamsLastUpdate == nil || now.Sub(*s.teamsLastUpdate).Hours() > teamsExpiry {
		teams, err := s.TBA.GetTeams(ctx)
		if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
			return nil
		} else if err != nil {
			return errors.Wrap(err, "retrieving teams")
		}

		if err := s.Store.TeamsUpsert(ctx, teams); err != nil {
			return errors.Wrap(err, "upserting teams")
		}

		s.teamsLastUpdate = &now
	}

	return nil
}
