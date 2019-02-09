package server

import (
	"context"
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type team struct {
	Rank         *int     `json:"rank,omitempty"`
	RankingScore *float64 `json:"rankingScore,omitempty"`
}

// teamsHandler returns a handler to get all teams at a given event.
func (s *Server) teamsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := mux.Vars(r)["eventKey"]

		event, err := s.Store.GetEvent(r.Context(), eventKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		}
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		// Get new team data from TBA
		if err := s.updateTeamKeys(r.Context(), eventKey); err != nil {
			// 404 if eventKey isn't a real event
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("updating team key data")
			return
		}

		teamKeys, err := s.Store.GetTeamKeys(r.Context(), eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving team key data")
			return
		}

		ihttp.Respond(w, teamKeys, http.StatusOK)
	}
}

// teamInfoHandler returns a handler to get info about a specific team at a specific event.
func (s *Server) teamInfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey, teamKey := vars["eventKey"], vars["teamKey"]

		event, err := s.Store.GetEvent(r.Context(), eventKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		}
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		// Get new team rankings data from TBA
		if err := s.updateTeamRankings(r.Context(), eventKey); err != nil {
			// 404 if eventKey isn't a real event
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("updating team rankings data")
			return
		}

		fullTeam, err := s.Store.GetTeam(r.Context(), teamKey, eventKey)
		if err != nil {
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving team rankings data")
			return
		}

		team := team{
			Rank:         fullTeam.Rank,
			RankingScore: fullTeam.RankingScore,
		}

		ihttp.Respond(w, team, http.StatusOK)
	}
}

// teamReportHandler returns a handler to get all reports about a specific team across all events.
func (s *Server) teamReportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		teamKey := vars["teamKey"]

		realmID, err := ihttp.GetRealmID(r)

		var reports []store.Report

		if err != nil {
			reports, err = s.Store.GetTeamReports(r.Context(), teamKey, nil)
		} else {
			reports, err = s.Store.GetTeamReports(r.Context(), teamKey, &realmID)
		}

		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving all-event team reports")
			return
		}

		ihttp.Respond(w, reports, http.StatusOK)
	}
}

// Get new team key data from TBA for a particular event. Upsert data into database.
func (s *Server) updateTeamKeys(ctx context.Context, eventKey string) error {
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

	return s.Store.TeamKeysUpsert(ctx, eventKey, teams)
}

// Get new team rankings data from TBA for a particular event. Upsert data into database.
func (s *Server) updateTeamRankings(ctx context.Context, eventKey string) error {
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

	return s.Store.TeamsUpsert(ctx, teams)
}
