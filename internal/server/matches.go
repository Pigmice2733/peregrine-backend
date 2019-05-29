package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/fharding1/gemux"
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
}

// matchesHandler returns a handler to get all matches at a given event.
func (s *Server) matchesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := gemux.PathParameter(r.Context(), 0)
		teams := r.URL.Query()["team"]
		tbaDeleted := r.URL.Query().Get("tbaDeleted") == "true"

		event, err := s.Store.GetEvent(r.Context(), eventKey)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		if err := s.updateMatches(r.Context(), event); err != nil {
			// 404 if eventKey isn't a real event
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("updating match data")
			return
		}

		fullMatches, err := s.Store.GetMatches(r.Context(), eventKey, teams, tbaDeleted)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event matches")
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
			})
		}

		ihttp.Respond(w, matches, http.StatusOK)
	}
}

// matchHandler returns a handler to get a specific match.
func (s *Server) matchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := gemux.PathParameter(r.Context(), 0)
		matchKey := gemux.PathParameter(r.Context(), 1)

		event, err := s.Store.GetEvent(r.Context(), eventKey)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey = fmt.Sprintf("%s_%s", eventKey, matchKey)

		if err := s.updateMatches(r.Context(), event); err != nil {
			// 404 if eventKey isn't a real event
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("updating match data")
			return
		}

		fullMatch, err := s.Store.GetMatch(r.Context(), matchKey)
		if err != nil {
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving match")
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

		eventKey := gemux.PathParameter(r.Context(), 0)

		event, err := s.Store.GetEvent(r.Context(), eventKey)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		m.Key = fmt.Sprintf("%s_%s", eventKey, m.Key)

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

		if err := s.Store.UpsertMatch(r.Context(), sm); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("upserting match")
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// Get new match data from TBA for a particular event. Upsert match data into database.
func (s *Server) updateMatches(ctx context.Context, event store.Event) error {
	// Check that eventKey is a valid event key
	valid, err := s.Store.CheckTBAEventKeyExists(ctx, event.Key)
	if err != nil {
		return errors.Wrap(err, "unable to check if tba event key exists")
	}
	if !valid {
		return nil
	}

	var fullMatches []store.Match

	if !event.TBADeleted {
		fullMatches, err = s.TBA.GetMatches(ctx, event.Key)
		if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
			return nil
		} else if err != nil {
			return errors.Wrap(err, "unable to fetch matches from TBA")
		}

		if err := s.Store.UpdateTBAMatches(ctx, event.Key, fullMatches); err != nil {
			return errors.Wrap(err, "unable to update matches")
		}
	}

	err = s.Store.MarkMatchesDeleted(ctx, event.Key, fullMatches)
	return errors.Wrap(err, "unable to mark deleted matches")
}
