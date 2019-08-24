package server

import (
	"fmt"
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/analysis"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
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

		analyzedStats, err := analysis.AnalyzeReports(schema, reports, teamKeys)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("analyzing event data")
			return
		}

		ihttp.Respond(w, analyzedStats, http.StatusOK)
	}
}
