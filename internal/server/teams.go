package server

import (
	"net/http"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

type team struct {
	NextMatch    *match   `json:"nextMatch,omitempty"`
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
			if _, ok := err.(store.ErrNoResults); ok {
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

		if teamKeys == nil {
			teamKeys = []string{}
		}

		ihttp.Respond(w, teamKeys, http.StatusOK)
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
			if _, ok := err.(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: updating team rankings data: %v\n", err)
			return
		}

		fullTeam, err := s.store.GetTeam(teamKey, eventKey)
		if err != nil {
			if _, ok := err.(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: retrieving team rankings data: %v\n", err)
			return
		}

		fullMatches, err := s.store.GetTeamMatches(eventKey, teamKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: retrieving team match data: %v\n", err)
			return
		}

		now := time.Now().Unix()
		var fullNextMatch *store.Match
		for i, fullMatch := range fullMatches {
			matchTime := fullMatch.GetTime()
			if fullNextMatch != nil {
				nextMatchTime := fullNextMatch.GetTime()
				if matchTime != nil && matchTime.Unix > now && matchTime.Unix < nextMatchTime.Unix {
					fullNextMatch = &fullMatches[i]
				}
			} else {
				if matchTime != nil && matchTime.Unix > now {
					fullNextMatch = &fullMatches[i]
				}
			}
		}

		var nextMatch *match
		if fullNextMatch != nil {
			nextMatch = &match{
				Key:          fullNextMatch.Key,
				Time:         fullNextMatch.GetTime(),
				RedScore:     fullNextMatch.RedScore,
				BlueScore:    fullNextMatch.BlueScore,
				RedAlliance:  fullNextMatch.RedAlliance,
				BlueAlliance: fullNextMatch.BlueAlliance,
			}
		}

		team := team{
			NextMatch:    nextMatch,
			Rank:         fullTeam.Rank,
			RankingScore: fullTeam.RankingScore,
		}

		ihttp.Respond(w, team, http.StatusOK)
	}
}

// Get new team key data from TBA for a particular event. Upsert data into database.
func (s *Server) updateTeamKeys(eventKey string) error {
	// Check that eventKey is a valid event key
	err := s.store.CheckEventKeyExists(eventKey)
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
	err := s.store.CheckEventKeyExists(eventKey)
	if err != nil {
		return err
	}

	teams, err := s.tba.GetTeamRankings(eventKey)
	if err != nil {
		return err
	}
	return s.store.TeamsUpsert(teams)
}
