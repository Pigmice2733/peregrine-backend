package server

import (
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

type team struct {
	Rank         *int     `json:"rank,omitempty"`
	RankingScore *float64 `json:"rankingScore,omitempty"`
}

// teamsHandler returns a handler to get all teams at a given event.
func (s *Server) teamsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := mux.Vars(r)["eventKey"]

		// Get new team data from TBA
		if err := s.updateTeamKeys(eventKey); err != nil {
			// 404 if eventKey isn't a real event
			if ok := store.IsNoResultError(err); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: updating team key data: %v\n", err)
			return
		}

		teamKeys, err := s.store.GetTeamKeys(eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: retrieving team key data: %v\n", err)
			return
		}

		ihttp.Respond(w, teamKeys, nil, http.StatusOK)
	}
}

// teamInfoHandler returns a handler to get info about a specific team at a specific event.
func (s *Server) teamInfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey, teamKey := vars["eventKey"], vars["teamKey"]

		// Get new team rankings data from TBA
		if err := s.updateTeamRankings(eventKey); err != nil {
			// 404 if eventKey isn't a real event
			if ok := store.IsNoResultError(err); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: updating team rankings data: %v\n", err)
			return
		}

		fullTeam, err := s.store.GetTeam(teamKey, eventKey)
		if err != nil {
			if ok := store.IsNoResultError(err); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: retrieving team rankings data: %v\n", err)
			return
		}

		team := team{
			Rank:         fullTeam.Rank,
			RankingScore: fullTeam.RankingScore,
		}

		ihttp.Respond(w, team, nil, http.StatusOK)
	}
}

// Get new team key data from TBA for a particular event. Upsert data into database.
func (s *Server) updateTeamKeys(eventKey string) error {
	// Check that eventKey is a valid event key
	err := s.store.CheckEventKey(eventKey)
	if err != nil {
		return err
	}

	teams, err := s.tba.GetTeamKeys(eventKey)
	if err != nil {
		return err
	}
	return s.store.TeamKeysUpsert(eventKey, teams)
}

// Get new team rankings data from TBA for a particular event. Upsert data into database.
func (s *Server) updateTeamRankings(eventKey string) error {
	// Check that eventKey is a valid event key
	err := s.store.CheckEventKey(eventKey)
	if err != nil {
		return err
	}

	teams, err := s.tba.GetTeamRankings(eventKey)
	if err != nil {
		return err
	}
	return s.store.TeamsUpsert(teams)
}
