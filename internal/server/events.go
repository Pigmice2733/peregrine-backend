package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// eventsHandler returns a handler to get all events in a given year.
func (s *Server) eventsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get new event data from TBA if event data is over 24 hours old
		if err := s.updateEvents(r.Context()); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("unable to update event data")
			return
		}

		var events []store.Event

		roles := ihttp.GetRoles(r)

		userRealm, getRealmErr := ihttp.GetRealmID(r)

		var err error
		if roles.IsSuperAdmin {
			events, err = s.Store.GetEvents(r.Context())
		} else {
			if getRealmErr != nil {
				events, err = s.Store.GetEventsFromRealm(r.Context(), nil)
			} else {
				events, err = s.Store.GetEventsFromRealm(r.Context(), &userRealm)
			}
		}

		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event data")
			return
		}

		ihttp.Respond(w, &events, http.StatusOK)
	}
}

// eventHandler returns a handler to get a specific event.
func (s *Server) eventHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get new event data from TBA if event data is over 24 hours old
		if err := s.updateEvents(r.Context()); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("unable to update event data")
			return
		}

		eventKey := mux.Vars(r)["eventKey"]

		event, err := s.Store.GetEvent(r.Context(), eventKey)
		if err != nil {
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("unable to retrieve event data")
			return
		}

		if !s.checkEventAccess(event.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
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
				go s.Logger.WithError(err).Error("unable to upsert event data")
				ihttp.Error(w, http.StatusInternalServerError)
			}

			return
		}

		if created {
			ihttp.Respond(w, nil, http.StatusCreated)
		} else {
			ihttp.Respond(w, nil, http.StatusNoContent)
		}
	}
}

const expiry = 3.0

// Get new event data from TBA only if event data is over 3 hours old.
// Upsert event data into database.
func (s *Server) updateEvents(ctx context.Context) error {
	now := time.Now()

	if s.eventsLastUpdate == nil || now.Sub(*s.eventsLastUpdate).Hours() > expiry {
		schema, err := s.Store.GetSchemaByYear(ctx, s.Year)
		var schemaID *int64
		if err != nil {
			_, ok := err.(store.ErrNoResults)
			if !ok {
				return err
			}
		} else {
			schemaID = &schema.ID
		}

		events, err := s.TBA.GetEvents(ctx, s.Year, schemaID)
		if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
			return nil
		} else if err != nil {
			return err
		}

		if err := s.Store.EventsUpsert(ctx, events); err != nil {
			return errors.Wrap(err, "upserting events")
		}

		s.eventsLastUpdate = &now
	}

	return nil
}

// Returns whether a user can access an event or its matches
func (s *Server) checkEventAccess(eventRealm *int64, r *http.Request) bool {
	if eventRealm == nil {
		return true
	}

	roles := ihttp.GetRoles(r)

	if roles.IsSuperAdmin {
		return true
	}

	userRealm, err := ihttp.GetRealmID(r)
	if err != nil {
		return false
	}
	return *eventRealm != userRealm
}
