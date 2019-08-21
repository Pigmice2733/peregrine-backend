package server

import (
	"fmt"
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/summary"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// eventStats analyzes the event-wide statistics of every team at an event with submitted reports
func (s *Server) eventStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]

		event, err := s.Store.GetEvent(r.Context(), eventKey)
		if _, ok := err.(store.ErrNoResults); ok {
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

		if event.SchemaID == nil {
			ihttp.Respond(w, fmt.Errorf("no schema found"), http.StatusBadRequest)
			return
		}

		// Get new team data from TBA
		if err := s.updateEventTeamKeys(r.Context(), eventKey); err != nil {
			// 404 if eventKey isn't a real event
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("updating team key data")
			return
		}

		var reports []store.Report

		realmID, err := ihttp.GetRealmID(r)

		if err != nil {
			reports, err = s.Store.GetEventReports(r.Context(), eventKey, nil)
		} else {
			reports, err = s.Store.GetEventReports(r.Context(), eventKey, &realmID)
		}

		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		schema, err := s.Store.GetSchemaByID(r.Context(), *event.SchemaID)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event schema")
			return
		}

		teamKeys, err := s.Store.GetEventTeamKeys(r.Context(), eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving team key data")
			return
		}

		matches, err := s.Store.GetAnalysisInfo(r.Context(), eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving match analysis info")
			return
		}

		teamAnalyses := make([]teamAnalysis, 0)

		_ = matches
		_ = teamKeys
		_ = reports
		_ = schema

		ihttp.Respond(w, teamAnalyses, http.StatusOK)
	}
}

type teamAnalysis struct {
	Team   string          `json:"team"`
	Auto   summary.Summary `json:"auto"`
	Teleop summary.Summary `json:"teleop"`
}
