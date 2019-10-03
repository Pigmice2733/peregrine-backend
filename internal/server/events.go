package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"errors"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

// eventsHandler returns a handler to get all events in a given year.
func (s *Server) eventsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tbaDeleted, _ := strconv.ParseBool(r.URL.Query().Get("tbaDeleted"))

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		events, err := s.Store.GetEventsForRealm(r.Context(), tbaDeleted, realmID)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event data")
			return
		}

		ihttp.Respond(w, &events, http.StatusOK)
	}
}

// eventHandler returns a handler to get a specific event.
func (s *Server) eventHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := mux.Vars(r)["eventKey"]

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		event, err := s.Store.GetEventForRealm(r.Context(), eventKey, realmID)
		if errors.Is(err, store.ErrNoResults{}) {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("unable to retrieve event data")
			return
		}

		ihttp.Respond(w, &event, http.StatusOK)
	}
}

func (s *Server) upsertEventHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := mux.Vars(r)["eventKey"]

		var event store.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		creatorRealm, err := ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}
		roles := ihttp.GetRoles(r)

		event.Key = eventKey
		event.RealmID = &creatorRealm

		existed, err := editEvent(r.Context(), s.Store, roles, creatorRealm, event.Key, func(tx *sqlx.Tx) error {
			if err := s.Store.UpsertEventTx(r.Context(), tx, event); err != nil {
				return fmt.Errorf("unable to upsert event: %w", err)
			}

			return nil
		})
		if errors.Is(err, forbiddenError{}) {
			ihttp.Error(w, http.StatusForbidden)
			return
		} else if err != nil {
			s.Logger.WithError(err).Error("unable to upsert event")
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

func editEvent(ctx context.Context, sto *store.Service, roles store.Roles, userRealmID int64, eventKey string, editFunc func(tx *sqlx.Tx) error) (existed bool, err error) {
	existed = true

	err = sto.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		if err := sto.ExclusiveLockEventsTx(ctx, tx); err != nil {
			return err
		}

		realmID, err := sto.GetEventRealmIDTx(ctx, tx, eventKey)
		if errors.Is(err, store.ErrNoResults{}) {
			existed = false
		} else if err != nil {
			return fmt.Errorf("unable to get events: %w", err)
		}

		if realmID == nil && !roles.IsSuperAdmin {
			return forbiddenError{errors.New("only super-admins can edit events with no realm ID")}
		} else if realmID != nil && *realmID != userRealmID {
			return forbiddenError{errors.New("only realm admins with matching realm IDs can edit events with a specified realm ID")}
		}

		if err := editFunc(tx); err != nil {
			return fmt.Errorf("unable to edit event: %w", err)
		}

		return nil
	})

	return existed, err
}
