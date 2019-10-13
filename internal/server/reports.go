package server

import (
	"encoding/json"
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/store"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

func (s *Server) getReports() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		matchKey := vars["matchKey"]
		teamKey := vars["teamKey"]

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		reports, err := s.Store.GetMatchTeamReportsForRealm(r.Context(), eventKey, matchKey, teamKey, realmID)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("getting reports")
			return
		}

		ihttp.Respond(w, reports, http.StatusOK)
	}
}

func (s *Server) putReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		matchKey := vars["matchKey"]
		teamKey := vars["teamKey"]

		var report store.Report
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		report.EventKey = eventKey
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
			s.Logger.WithError(err).Error("upserting report")
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
		realmID, err := ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		leaderboard, err := s.Store.GetLeaderboardForRealm(r.Context(), realmID)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("getting leaderboard")
			return
		}

		ihttp.Respond(w, leaderboard, http.StatusOK)
	}
}
