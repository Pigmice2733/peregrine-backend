package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type match struct {
	Key           string     `json:"key"`
	Time          *time.Time `json:"time"`
	ScheduledTime *time.Time `json:"scheduledTime,omitempty"`
	RedScore      *int       `json:"redScore,omitempty"`
	BlueScore     *int       `json:"blueScore,omitempty"`
	RedAlliance   []string   `json:"redAlliance"`
	BlueAlliance  []string   `json:"blueAlliance"`
	TBADeleted    bool       `json:"tbaDeleted"`
	TBAURL        *string    `json:"tbaUrl"`
}

// matchesHandler returns a handler to get all matches at a given event.
func (s *Server) matchesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := mux.Vars(r)["eventKey"]
		teams := r.URL.Query()["team"]
		tbaDeleted := r.URL.Query().Get("tbaDeleted") == "true"

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

		fullMatches, err := s.Store.GetMatches(r.Context(), eventKey, teams, tbaDeleted)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event matches")
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
				TBADeleted:    fullMatch.TBADeleted,
				TBAURL:        fullMatch.TBAURL,
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

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey = fmt.Sprintf("%s_%s", eventKey, matchKey)

		fullMatch, err := s.Store.GetMatch(r.Context(), matchKey)
		if err != nil {
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving match")
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
			TBADeleted:   fullMatch.TBADeleted,
			TBAURL:       fullMatch.TBAURL,
		}

		ihttp.Respond(w, match, http.StatusOK)
	}
}

func (s *Server) upsertMatchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m match
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		if m.RedAlliance == nil || m.BlueAlliance == nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		matchKey := vars["matchKey"]

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

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey = fmt.Sprintf("%s_%s", eventKey, matchKey)

		sm := store.Match{
			Key:           matchKey,
			EventKey:      eventKey,
			ActualTime:    m.Time,
			ScheduledTime: m.Time,
			RedScore:      m.RedScore,
			BlueScore:     m.BlueScore,
			RedAlliance:   m.RedAlliance,
			BlueAlliance:  m.BlueAlliance,
			TBAURL:        m.TBAURL,
		}

		if err := s.Store.UpsertMatch(r.Context(), sm); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("upserting match")
			return
		}

		ihttp.Respond(w, m, http.StatusOK)
	}
}

func (s *Server) deleteMatchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		matchKey := vars["matchKey"]

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

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey = fmt.Sprintf("%s_%s", eventKey, matchKey)

		err = s.Store.DeleteMatch(r.Context(), matchKey)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("upserting match")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
