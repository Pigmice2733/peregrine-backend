package server

import (
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/analysis"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type teamStats struct {
	Team   string                  `json:"team"`
	Auto   []analysis.AnalyzedStat `json:"auto"`
	Teleop []analysis.AnalyzedStat `json:"teleop"`
}

// eventTeamStats analyzes the event-wide statistics of every team at an event with submitted reports
func (s *Server) eventTeamStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		teamKey := vars["teamKey"]

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

		reports, err := s.Store.GetTeamEventReports(eventKey, teamKey)
		if _, ok := err.(*store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		schema, err := s.Store.GetSchemaByID(*event.SchemaID)
		if _, ok := err.(*store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusConflict)
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

		fullStats := []teamStats{}

		for team, ts := range analyzedStats {
			stats := teamStats{
				Team:   team,
				Auto:   []analysis.AnalyzedStat{},
				Teleop: []analysis.AnalyzedStat{},
			}

			for _, stat := range ts.Auto {
				stats.Auto = append(stats.Auto, stat)
			}

			for _, stat := range ts.Teleop {
				stats.Teleop = append(stats.Auto, stat)
			}

			fullStats = append(fullStats, stats)
		}

		teamKeys, err := s.Store.GetTeamKeys(r.Context(), eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving team key data")
			return
		}

		for _, team := range teamKeys {
			if _, ok := analyzedStats[team]; !ok {
				stats := teamStats{
					Team:   team,
					Auto:   []analysis.AnalyzedStat{},
					Teleop: []analysis.AnalyzedStat{},
				}
				fullStats = append(fullStats, stats)
			}
		}

		ihttp.Respond(w, fullStats, http.StatusOK)
	}
}
