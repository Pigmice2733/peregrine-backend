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

		autoSchema := storeStatDescsToSummarySchema(schema.Auto)
		teleopSchema := storeStatDescsToSummarySchema(schema.Teleop)

		teamAnalyses := make([]teamAnalysis, 0)

		for _, team := range teamKeys {
			autoReports, teleopReports := storeReportsToSummaryReports(reports, team)
			stats := storeMatchToSummaryStats(matches, team)

			ta := teamAnalysis{Team: team}

			ta.Auto, err = summary.Summarize(autoReports, stats, autoSchema)
			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				s.Logger.WithError(err).Error("analyzing event data")
				return
			}

			ta.Teleop, err = summary.Summarize(teleopReports, stats, teleopSchema)
			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				s.Logger.WithError(err).Error("analyzing event data")
				return
			}

			teamAnalyses = append(teamAnalyses, ta)
		}

		ihttp.Respond(w, teamAnalyses, http.StatusOK)
	}
}

type teamAnalysis struct {
	Team   string          `json:"team"`
	Auto   summary.Summary `json:"auto"`
	Teleop summary.Summary `json:"teleop"`
}

func storeMatchToSummaryStats(matches []store.Match, team string) summary.EventStats {
	stats := make(summary.EventStats)

	for _, match := range matches {
		breakdown := match.RedScoreBreakdown
		robotIndex := indexOf(match.RedAlliance, team)
		if robotIndex == -1 {
			breakdown = match.BlueScoreBreakdown
			robotIndex = indexOf(match.BlueAlliance, team)
		}

		if robotIndex == -1 {
			continue
		}

		stats[match.Key] = summary.MatchStats{
			RobotIndex: robotIndex,
			Stats:      breakdown,
		}
	}

	return stats
}

func storeStatDescsToSummarySchema(descs store.StatDescriptions) summary.Schema {
	var schema summary.Schema

	for _, desc := range descs {
		sum := make([]summary.Reference, 0)
		for _, v := range desc.Sum {
			sum = append(sum, summary.Reference{TBA: v.TBA, Field: v.Field})
		}

		anyOf := make([]summary.AnyOf, 0)
		for _, v := range desc.AnyOf {
			anyOf = append(anyOf, summary.AnyOf{Reference: summary.Reference{TBA: v.TBA, Field: v.Field}, Equals: v.Equals})
		}

		schema = append(schema, summary.SchemaStat{
			Name:  desc.Name,
			Type:  desc.Type,
			AnyOf: anyOf,
			Sum:   sum,
		})
	}

	return schema
}

func storeReportsToSummaryReports(reports []store.Report, team string) (auto, teleop summary.EventReports) {
	autoReports := make(summary.EventReports)
	teleopReports := make(summary.EventReports)

	for _, report := range reports {
		if report.TeamKey == team {
			autoReports[report.MatchKey] = make(summary.MatchTeamReport)
			teleopReports[report.MatchKey] = make(summary.MatchTeamReport)

			for _, stat := range report.Data.Auto {
				autoReports[report.MatchKey][stat.Name] = stat.Value
			}

			for _, stat := range report.Data.Teleop {
				teleopReports[report.MatchKey][stat.Name] = stat.Value
			}
		}
	}

	return autoReports, teleopReports
}

func indexOf(arr []string, v string) int {
	for i, b := range arr {
		if b == v {
			return i
		}
	}

	return -1
}
