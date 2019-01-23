package server

import (
	"fmt"
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/analysis"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type teamStats struct {
	Team   string        `json:"team"`
	Auto   []interface{} `json:"auto"`
	Teleop []interface{} `json:"teleop"`
}

// eventStats analyzes the event-wide statistics of every team at an event with submitted reports
func (s *Server) eventStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]

		event, err := s.Store.GetEvent(r.Context(), eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event")
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

		var reports []store.Report

		realmID, err := ihttp.GetRealmID(r)

		if err != nil {
			reports, err = s.Store.GetEventReports(r.Context(), eventKey, nil)
		} else {
			reports, err = s.Store.GetEventReports(r.Context(), eventKey, &realmID)
		}

		if _, ok := err.(*store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		schema, err := s.Store.GetSchemaByID(r.Context(), *event.SchemaID)
		if _, ok := err.(*store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event schema")
			return
		}

		analyzedStats, err := analysis.AnalyzeReports(schema, reports)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("analyzing event data")
			return
		}

		fullStats := make([]teamStats, 0)
		for _, ts := range analyzedStats {
			stats := teamStats{Team: ts.Team, Auto: make([]interface{}, 0), Teleop: make([]interface{}, 0)}

			for _, v := range ts.AutoBoolean {
				stats.Auto = append(stats.Auto, v)
			}
			for _, v := range ts.AutoNumeric {
				stats.Auto = append(stats.Auto, v)
			}
			for _, v := range ts.TeleopBoolean {
				stats.Teleop = append(stats.Teleop, v)
			}
			for _, v := range ts.TeleopNumeric {
				stats.Teleop = append(stats.Teleop, v)
			}

			fullStats = append(fullStats, stats)
		}

		teamKeys, err := s.Store.GetTeamKeys(r.Context(), eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving team key data")
			return
		}

		// fill in unreported teams
		for _, team := range teamKeys {
			if _, ok := analyzedStats[team]; !ok {
				fullStats = append(fullStats, teamStats{Team: team, Auto: make([]interface{}, 0), Teleop: make([]interface{}, 0)})
			}
		}

		ihttp.Respond(w, fullStats, http.StatusOK)
	}
}
