package server

import (
	"encoding/json"
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/analysis"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type teamStats struct {
	Team   string            `json:"team"`
	Auto   []json.RawMessage `json:"auto"`
	Teleop []json.RawMessage `json:"teleop"`
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
			reports, err = s.Store.GetEventReports(eventKey, nil)
		} else {
			reports, err = s.Store.GetEventReports(eventKey, &realmID)
		}

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

		for _, ts := range analyzedStats {
			stats, err := marshalTeamStats(ts)

			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				go s.Logger.WithError(err).Error("marshalling statistic")
				return
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
					Auto:   []json.RawMessage{},
					Teleop: []json.RawMessage{},
				}
				fullStats = append(fullStats, stats)
			}
		}

		ihttp.Respond(w, fullStats, http.StatusOK)
	}
}

func marshalTeamStats(ts *analysis.TeamStats) (teamStats, error) {
	stats := teamStats{
		Team:   ts.Team,
		Auto:   []json.RawMessage{},
		Teleop: []json.RawMessage{},
	}

	for _, numeric := range ts.AutoNumeric {
		stat, err := json.Marshal(numeric)
		if err != nil {
			return stats, err
		}
		stats.Auto = append(stats.Auto, stat)
	}

	for _, boolean := range ts.AutoBoolean {
		stat, err := json.Marshal(boolean)
		if err != nil {
			return stats, err
		}
		stats.Auto = append(stats.Auto, stat)
	}

	for _, numeric := range ts.TeleopNumeric {
		stat, err := json.Marshal(numeric)
		if err != nil {
			return stats, err
		}
		stats.Teleop = append(stats.Teleop, stat)
	}

	for _, boolean := range ts.TeleopBoolean {
		stat, err := json.Marshal(boolean)
		if err != nil {
			return stats, err
		}
		stats.Teleop = append(stats.Teleop, stat)
	}

	return stats, nil
}
