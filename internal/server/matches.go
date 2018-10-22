package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
)

type match struct {
	Key           string          `json:"key"`
	Time          *store.UnixTime `json:"time"`
	ScheduledTime *store.UnixTime `json:"scheduledTime,omitempty"`
	RedScore      *int            `json:"redScore,omitempty"`
	BlueScore     *int            `json:"blueScore,omitempty"`
	RedAlliance   []string        `json:"redAlliance"`
	BlueAlliance  []string        `json:"blueAlliance"`
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
			go s.logger.WithError(err).Error("updating match data")
			return
		}

		fullMatches, err := s.store.GetEventMatches(eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.logger.WithError(err).Error("retrieving event matches")
			return
		}

		matches := []match{}
		for _, fullMatch := range fullMatches {
			// Match keys are stored in TBA format, with a leading event key
			// prefix that which needs to be removed before use.
			key := strings.TrimPrefix(fullMatch.Key, eventKey+"_")
			matches = append(matches, match{
				Key:           key,
				Time:          fullMatch.GetTime(),
				ScheduledTime: fullMatch.ScheduledTime,
				RedScore:      fullMatch.RedScore,
				BlueScore:     fullMatch.BlueScore,
				RedAlliance:   fullMatch.RedAlliance,
				BlueAlliance:  fullMatch.BlueAlliance,
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

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey = fmt.Sprintf("%s_%s", eventKey, matchKey)

		// Get new match data from TBA
		if err := s.updateMatches(eventKey); err != nil {
			// 404 if eventKey isn't a real event
			if _, ok := err.(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.logger.WithError(err).Error("updating match data")
			return
		}

		fullMatch, err := s.store.GetMatch(matchKey)
		if err != nil {
			if _, ok := err.(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.logger.WithError(err).Error("retrieving match")
			return
		}

		// Match keys are stored in TBA format, with a leading event key
		// prefix that which needs to be removed before use.
		key := strings.TrimPrefix(fullMatch.Key, eventKey+"_")
		match := match{
			Key:          key,
			Time:         fullMatch.GetTime(),
			RedScore:     fullMatch.RedScore,
			BlueScore:    fullMatch.BlueScore,
			RedAlliance:  fullMatch.RedAlliance,
			BlueAlliance: fullMatch.BlueAlliance,
		}

		ihttp.Respond(w, match, http.StatusOK)
	}
}

func (s *Server) createMatchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m match
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		eventKey := mux.Vars(r)["eventKey"]

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		m.Key = fmt.Sprintf("%s_%s", eventKey, m.Key)

		// this is redundant since the route should be admin-protected anyways
		if !ihttp.GetRoles(r).IsAdmin {
			ihttp.Error(w, http.StatusForbidden)
			go s.logger.Error("got non-admin user on admin-protected route")
			return
		}

		sm := store.Match{
			Key:           m.Key,
			EventKey:      eventKey,
			ActualTime:    m.Time,
			ScheduledTime: m.Time,
			RedScore:      m.RedScore,
			BlueScore:     m.BlueScore,
			RedAlliance:   m.RedAlliance,
			BlueAlliance:  m.BlueAlliance,
		}

		if err := s.store.MatchesUpsert([]store.Match{sm}); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.logger.WithError(err).Error("upserting match")
			return
		}

		ihttp.Respond(w, nil, http.StatusCreated)
	}
}

// Get new match data from TBA for a particular event. Upsert match data into database.
func (s *Server) updateMatches(eventKey string) error {
	// Check that eventKey is a valid event key
	err := s.store.CheckTBAEventKeyExists(eventKey)
	if err == store.ErrManuallyAdded {
		return nil
	} else if err != nil {
		return err
	}

	fullMatches, err := s.tba.GetMatches(eventKey)
	if err != nil {
		return err
	}

	return s.store.MatchesUpsert(fullMatches)
}
