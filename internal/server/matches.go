package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"errors"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
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
	Videos        []string   `json:"videos"`
}

// matchesHandler returns a handler to get all matches at a given event.
func (s *Server) matchesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := mux.Vars(r)["eventKey"]
		teams := r.URL.Query()["team"]
		tbaDeleted, _ := strconv.ParseBool(r.URL.Query().Get("tbaDeleted"))

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		fullMatches, err := s.Store.GetMatchesForRealm(r.Context(), eventKey, teams, tbaDeleted, realmID)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event matches")
			return
		}

		matches := []match{}
		for _, fullMatch := range fullMatches {
			matches = append(matches, match{
				Key:           fullMatch.Key,
				Time:          fullMatch.GetTime(),
				ScheduledTime: fullMatch.ScheduledTime,
				RedScore:      fullMatch.RedScore,
				BlueScore:     fullMatch.BlueScore,
				RedAlliance:   fullMatch.RedAlliance,
				BlueAlliance:  fullMatch.BlueAlliance,
				TBADeleted:    fullMatch.TBADeleted,
				TBAURL:        fullMatch.TBAURL,
				Videos:        fullMatch.Videos,
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

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		fullMatch, err := s.Store.GetMatchForRealm(r.Context(), eventKey, matchKey, realmID)
		if errors.Is(err, store.ErrNoResults{}) {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving match")
			return
		}

		match := match{
			Key:          fullMatch.Key,
			Time:         fullMatch.GetTime(),
			RedScore:     fullMatch.RedScore,
			BlueScore:    fullMatch.BlueScore,
			RedAlliance:  fullMatch.RedAlliance,
			BlueAlliance: fullMatch.BlueAlliance,
			TBADeleted:   fullMatch.TBADeleted,
			TBAURL:       fullMatch.TBAURL,
			Videos:       fullMatch.Videos,
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
			Videos:        m.Videos,
		}

		roles := ihttp.GetRoles(r)

		userRealmID, err := ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}

		existed, err := editMatch(r.Context(), s.Store, roles, userRealmID, sm.Key, func(tx *sqlx.Tx) error {
			if err := s.Store.UpsertMatchTx(r.Context(), tx, sm); err != nil {
				return fmt.Errorf("unable to upsert match: %w", err)
			}

			return nil
		})
		if errors.Is(err, forbiddenError{}) {
			ihttp.Error(w, http.StatusForbidden)
			return
		} else if err != nil {
			s.Logger.WithError(err).Error("unable to upsert match")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		if existed {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	}
}

func (s *Server) deleteMatchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		matchKey := vars["matchKey"]

		roles := ihttp.GetRoles(r)

		userRealmID, err := ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}

		existed, err := editMatch(r.Context(), s.Store, roles, userRealmID, matchKey, func(tx *sqlx.Tx) error {
			if err := s.Store.DeleteMatchTx(r.Context(), tx, matchKey, eventKey); err != nil {
				return fmt.Errorf("unable to delete match: %w", err)
			}

			return nil
		})
		if errors.Is(err, forbiddenError{}) {
			ihttp.Error(w, http.StatusForbidden)
			return
		} else if err != nil {
			s.Logger.WithError(err).Error("unable to delete match")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		if existed {
			w.WriteHeader(http.StatusNoContent)
		} else {
			ihttp.Error(w, http.StatusNotFound)
		}
	}
}

func editMatch(ctx context.Context, sto *store.Service, roles store.Roles, userRealmID int64, matchKey string, editFunc func(tx *sqlx.Tx) error) (existed bool, err error) {
	existed = true

	err = sto.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		if err := sto.ExclusiveLockMatchesTx(ctx, tx); err != nil {
			return err
		}

		realmID, err := sto.GetEventRealmIDByMatchKeyTx(ctx, tx, matchKey)
		if errors.Is(err, store.ErrNoResults{}) {
			existed = false
		} else if err != nil {
			return fmt.Errorf("unable to get matches: %w", err)
		}

		if realmID == nil && !roles.IsSuperAdmin {
			return forbiddenError{errors.New("only super-admins can edit matches with no realm ID")}
		} else if realmID != nil && *realmID != userRealmID {
			return forbiddenError{errors.New("only realm admins with matching realm IDs can edit matches with a specified realm ID")}
		}

		if err := editFunc(tx); err != nil {
			return fmt.Errorf("unable to edit match: %w", err)
		}

		return nil
	})

	return existed, err
}
