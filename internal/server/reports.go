package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/fharding1/gemux"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

func (s *Server) getReports() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := gemux.PathParameter(r.Context(), 0)
		partialMatchKey := gemux.PathParameter(r.Context(), 1)
		teamKey := gemux.PathParameter(r.Context(), 2)

		if _, err := s.Store.CheckTBAEventKeyExists(r.Context(), eventKey); err != nil {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey := fmt.Sprintf("%s_%s", eventKey, partialMatchKey)

		exists, err := s.Store.CheckMatchKeyExists(matchKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("checking that match exists")
			return
		}
		if !exists {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		reports, err := s.Store.GetTeamMatchReports(r.Context(), matchKey, teamKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("getting reports")
			return
		}

		ihttp.Respond(w, reports, http.StatusOK)
	}
}

func (s *Server) putReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := gemux.PathParameter(r.Context(), 0)
		partialMatchKey := gemux.PathParameter(r.Context(), 1)
		teamKey := gemux.PathParameter(r.Context(), 2)

		if _, err := s.Store.CheckTBAEventKeyExists(r.Context(), eventKey); err != nil {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey := fmt.Sprintf("%s_%s", eventKey, partialMatchKey)

		exists, err := s.Store.CheckMatchKeyExists(matchKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("checking that match exists")
			return
		} else if !exists {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		var report store.Report
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		report.MatchKey = matchKey
		report.TeamKey = teamKey

		reporterID, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}
		report.ReporterID = &reporterID

		var realmID int64
		realmID, err = ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}
		report.RealmID = &realmID

		created, err := s.Store.UpsertReport(r.Context(), report)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("upserting report")
			return
		}

		if created {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func (s *Server) leaderboardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leaderboard, err := s.Store.GetLeaderboard(r.Context())
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("getting leaderboard")
			return
		}

		ihttp.Respond(w, leaderboard, http.StatusOK)
	}
}
