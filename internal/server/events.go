package server

import (
	"encoding/json"
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
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

		var e store.Event
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		creatorRealm, err := ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		e.Key = eventKey
		e.RealmID = &creatorRealm

		created, err := s.Store.UpsertEvent(r.Context(), e)
		if err != nil {
			switch errors.Cause(err).(type) {
			case store.ErrFKeyViolation:
				ihttp.Error(w, http.StatusUnprocessableEntity)
			default:
				s.Logger.WithError(err).Error("unable to upsert event data")
				ihttp.Error(w, http.StatusInternalServerError)
			}

			return
		}

		if created {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
