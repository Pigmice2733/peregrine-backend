package server

import (
	"encoding/json"
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// eventsHandler returns a handler to get all events in a given year.
func (s *Server) eventsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tbaDeleted := r.URL.Query().Get("tbaDeleted") == "true"

		roles := ihttp.GetRoles(r)

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err != nil {
			realmID = &userRealmID
		}

		var events []store.Event
		if roles.IsSuperAdmin {
			events, err = s.Store.GetEvents(r.Context(), tbaDeleted)
		} else {
			events, err = s.Store.GetEventsForRealm(r.Context(), tbaDeleted, realmID)
		}

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
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
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

		var eventCreated bool

		err = s.Store.DoTransaction(r.Context(), func(tx *sqlx.Tx) error {
			if err := s.Store.ExclusiveLockEventsTx(r.Context(), tx); err != nil {
				s.Logger.WithError(err).Error("unable to acquire exclusive lock")
				ihttp.Error(w, http.StatusInternalServerError)
				return err
			}

			realmID, err := s.Store.GetEventRealmIDTx(r.Context(), tx, event.Key)
			if _, ok := err.(store.ErrNoResults); ok {
				eventCreated = true
			} else if err != nil {
				s.Logger.WithError(err).Error("unable to get event realm ID")
				ihttp.Error(w, http.StatusInternalServerError)
				return err
			}

			if realmID == nil && !roles.IsSuperAdmin {
				ihttp.Error(w, http.StatusForbidden)
				return errors.New("only super-admins can edit events with no realm ID")
			} else if realmID != nil && *realmID != creatorRealm && !roles.IsSuperAdmin {
				ihttp.Error(w, http.StatusForbidden)
				return errors.New("only super-admins or realm admins with matching realm IDs can edit events with a specified realm ID")
			}

			if err := s.Store.UpsertEventTx(r.Context(), tx, event); err != nil {
				s.Logger.WithError(err).Error("unable to upsert event data")
				ihttp.Error(w, http.StatusInternalServerError)
				return err
			}

			return nil
		})

		if err != nil {
			// responses are written within tx handler
			return
		}

		if eventCreated {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
