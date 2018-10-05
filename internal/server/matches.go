package server

import (
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

type match struct {
	Key          string          `json:"key"`
	Time         *store.UnixTime `json:"time"`
	RedScore     *int            `json:"redScore,omitempty"`
	BlueScore    *int            `json:"blueScore,omitempty"`
	RedAlliance  []string        `json:"redAlliance"`
	BlueAlliance []string        `json:"blueAlliance"`
}

// matchesHandler returns a handler to get all matches at a given event.
func (s *Server) matchesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := mux.Vars(r)["eventKey"]

		// Get new match data from TBA
		if err := s.updateMatches(eventKey); err != nil {
			// 404 if eventKey isn't a real event
			if _, ok := err.(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: updating match data: %v\n", err)
			return
		}

		fullMatches, err := s.store.GetEventMatches(eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: retrieving match data: %v\n", err)
			return
		}

		matches := []match{}
		for _, fullMatch := range fullMatches {
			matches = append(matches, match{
				Key:          fullMatch.Key,
				Time:         fullMatch.GetTime(),
				RedScore:     fullMatch.RedScore,
				BlueScore:    fullMatch.BlueScore,
				RedAlliance:  fullMatch.RedAlliance,
				BlueAlliance: fullMatch.BlueAlliance,
			})
		}

		ihttp.Respond(w, matches, http.StatusOK)
	}
}

// matchHandler returns a handler to get a specific match.
func (s *Server) matchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey, matchKey := vars["eventKey"], vars["matchKey"]

		// Get new match data from TBA
		if err := s.updateMatches(eventKey); err != nil {
			// 404 if eventKey isn't a real event
			if _, ok := err.(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: updating match data: %v\n", err)
			return
		}

		fullMatch, err := s.store.GetMatch(matchKey)
		if err != nil {
			if _, ok := err.(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: retrieving match data: %v\n", err)
			return
		}

		match := match{
			Key:          fullMatch.Key,
			Time:         fullMatch.GetTime(),
			RedScore:     fullMatch.RedScore,
			BlueScore:    fullMatch.BlueScore,
			RedAlliance:  fullMatch.RedAlliance,
			BlueAlliance: fullMatch.BlueAlliance,
		}

		ihttp.Respond(w, match, http.StatusOK)
	}
}

// Get new match data from TBA for a particular event. Upsert match data into database.
func (s *Server) updateMatches(eventKey string) error {
	// Check that eventKey is a valid event key
	err := s.store.CheckEventKeyExists(eventKey)
	if err != nil {
		return err
	}

	fullMatches, err := s.tba.GetMatches(eventKey)
	if err != nil {
		return err
	}
	return s.store.MatchesUpsert(fullMatches)
}
