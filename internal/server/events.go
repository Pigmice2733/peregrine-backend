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

type location struct {
	Name *string `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type event struct {
	Key          string         `json:"key"`
	RealmID      *int64         `json:"realmId,omitempty"`
	SchemaID     *int64         `json:"schemaId,omitempty"`
	Name         string         `json:"name"`
	District     *string        `json:"district,omitempty"`
	FullDistrict *string        `json:"fullDistrict,omitempty"`
	Week         *int           `json:"week,omitempty"`
	StartDate    store.UnixTime `json:"startDate"`
	EndDate      store.UnixTime `json:"endDate"`
	Location     location       `json:"location"`
}

type webcast struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type webcastEvent struct {
	event
	Webcasts []webcast `json:"webcasts"`
}

// eventsHandler returns a handler to get all events in a given year.
func (s *Server) eventsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get new event data from TBA if event data is over 24 hours old
		if err := s.updateEvents(r.Context()); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("unable to update event data")
			return
		}

		var fullEvents []store.Event

		roles := ihttp.GetRoles(r)

		userRealm, err := ihttp.GetRealmID(r)

		if roles.IsSuperAdmin {
			fullEvents, err = s.Store.GetEvents(r.Context())
		} else {
			if err != nil {
				fullEvents, err = s.Store.GetEventsFromRealm(r.Context(), nil)
			} else {
				fullEvents, err = s.Store.GetEventsFromRealm(r.Context(), &userRealm)
			}
		}

		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event data")
			return
		}

		events := []event{}
		for _, fullEvent := range fullEvents {
			events = append(events, event{
				Key:          fullEvent.Key,
				RealmID:      fullEvent.RealmID,
				SchemaID:     fullEvent.SchemaID,
				Name:         fullEvent.Name,
				District:     fullEvent.District,
				FullDistrict: fullEvent.FullDistrict,
				Week:         fullEvent.Week,
				StartDate:    fullEvent.StartDate,
				EndDate:      fullEvent.EndDate,
				Location: location{
					Lat: fullEvent.Location.Lat,
					Lon: fullEvent.Location.Lon,
				},
			})
		}

		ihttp.Respond(w, events, http.StatusOK)
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

		fullEvent, err := s.Store.GetEvent(r.Context(), eventKey)
		if err != nil {
			if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("unable to retrieve event data")
			return
		}

		if !s.checkEventAccess(fullEvent.RealmID, r) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		webcasts := []webcast{}
		for _, fullWebcast := range fullEvent.Webcasts {
			webcasts = append(webcasts, webcast{
				Type: string(fullWebcast.Type),
				URL:  fullWebcast.URL,
			})
		}

		event := webcastEvent{
			event: event{
				Key:          fullEvent.Key,
				RealmID:      fullEvent.RealmID,
				SchemaID:     fullEvent.SchemaID,
				Name:         fullEvent.Name,
				District:     fullEvent.District,
				FullDistrict: fullEvent.FullDistrict,
				Week:         fullEvent.Week,
				StartDate:    fullEvent.StartDate,
				EndDate:      fullEvent.EndDate,
				Location: location{
					Name: &fullEvent.Location.Name,
					Lat:  fullEvent.Location.Lat,
					Lon:  fullEvent.Location.Lon,
				},
			},
			Webcasts: webcasts,
		}

		// Using &event so that pointer receivers on embedded types get promoted
		ihttp.Respond(w, &event, http.StatusOK)
	}
}

func (s *Server) createEventHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var e store.Event
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		creatorRealm, err := ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}

		e.RealmID = &creatorRealm

		err = s.Store.EventsUpsert(r.Context(), []store.Event{e})

		if err != nil {
			switch errors.Cause(err).(type) {
			case store.ErrNoResults:
				ihttp.Error(w, http.StatusNotFound)
			case store.ErrFKeyViolation:
				ihttp.Error(w, http.StatusUnprocessableEntity)
			default:
				go s.Logger.WithError(err).Error("unable to upsert event data")
				ihttp.Error(w, http.StatusInternalServerError)
			}

			return
		}

		ihttp.Respond(w, nil, http.StatusCreated)
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
			_, ok := err.(*store.ErrNoResults)
			if !ok {
				return err
			}
		} else {
			schemaID = &schema.ID
		}

		fullEvents, err := s.TBA.GetEvents(ctx, s.Year, schemaID)
		if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
			return nil
		} else if err != nil {
			return err
		}

		if err := s.Store.EventsUpsert(ctx, fullEvents); err != nil {
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
