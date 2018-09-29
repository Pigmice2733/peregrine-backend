package server

import (
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

type match struct {
	ID           string          `json:"id"`
	Time         *store.UnixTime `json:"time"`
	RedScore     *int            `json:"redScore,omitempty"`
	BlueScore    *int            `json:"blueScore,omitempty"`
	RedAlliance  []string        `json:"redAlliance"`
	BlueAlliance []string        `json:"blueAlliance"`
}

// matchesHandler returns a handler to get all matches at a given event.
func (s *Server) matchesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]

		// Get new match data from TBA
		err := s.updateMatches(eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: updating match data: %v\n", err)
			return
		}

		fullMatches, err := s.store.GetEventMatches(eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Println(err)
			return
		}

		var matches []match
		for _, fullMatch := range fullMatches {
			var time *store.UnixTime

			if fullMatch.ActualTime != nil {
				time = fullMatch.ActualTime
			} else {
				time = fullMatch.PredictedTime
			}

			matches = append(matches, match{
				ID:           fullMatch.ID,
				Time:         time,
				RedScore:     fullMatch.RedScore,
				BlueScore:    fullMatch.BlueScore,
				RedAlliance:  fullMatch.RedAlliance,
				BlueAlliance: fullMatch.BlueAlliance,
			})
		}

		err = ihttp.Respond(w, matches, nil, http.StatusOK)
		if err != nil {
			s.logger.Println(err)
		}
	}
}

// matchHandler returns a handler to get a specific match.
func (s *Server) matchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey, matchKey := vars["eventKey"], vars["matchKey"]

		// Get new match data from TBA
		err := s.updateMatches(eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: updating match data: %v\n", err)
			return
		}

		fullMatch, err := s.store.GetMatch(matchKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Println(err)
			return
		}

		var time *store.UnixTime
		if fullMatch.ActualTime != nil {
			time = fullMatch.ActualTime
		} else {
			time = fullMatch.PredictedTime
		}

		match := match{
			ID:           fullMatch.ID,
			Time:         time,
			RedScore:     fullMatch.RedScore,
			BlueScore:    fullMatch.BlueScore,
			RedAlliance:  fullMatch.RedAlliance,
			BlueAlliance: fullMatch.BlueAlliance,
		}

		err = ihttp.Respond(w, match, nil, http.StatusOK)
		if err != nil {
			s.logger.Println(err)
		}
	}
}

// Get new match data from TBA for a particular event. Upsert match data into database.
func (s *Server) updateMatches(eventID string) error {
	fullMatches, err := s.tba.GetMatches(eventID)
	if err != nil {
		return err
	}
	return s.store.MatchesUpsert(fullMatches)
}
