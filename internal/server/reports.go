package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/store"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

func (s *Server) getReports() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		partialMatchKey := vars["matchKey"]
		teamKey := vars["teamKey"]

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
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		partialMatchKey := vars["matchKey"]
		teamKey := vars["teamKey"]

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

		var report store.Report
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		report.MatchKey = matchKey
		report.TeamKey = teamKey

		reporterID, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}
		report.ReporterID = &reporterID

		var realmID int64
		realmID, err = ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}
		report.RealmID = &realmID

		err = s.Store.UpsertReport(r.Context(), report)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("upserting report")
			return
		}

		ihttp.Respond(w, nil, http.StatusOK)
	}
}
