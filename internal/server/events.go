package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/gorilla/mux"
)

type location struct {
	Name *string `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type event struct {
	Key          string         `json:"key"`
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
		if err := s.updateEvents(); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("unable to update event data")
			return
		}

		fullEvents, err := s.Store.GetEvents()
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving event data")
			return
		}

		events := []event{}
		for _, fullEvent := range fullEvents {
			events = append(events, event{
				Key:          fullEvent.Key,
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
		if err := s.updateEvents(); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("unable to update event data")
			return
		}

		eventKey := mux.Vars(r)["eventKey"]

		fullEvent, err := s.Store.GetEvent(eventKey)
		if err != nil {
			if _, ok := err.(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("unable to retrieve event data")
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

		e.ManuallyAdded = true

		// this is redundant since the route should be admin-protected anyways
		if !ihttp.GetRoles(r).IsAdmin {
			ihttp.Error(w, http.StatusForbidden)
			go s.Logger.Error("got non-admin user on admin-protected route")
			return
		}

		if err := s.Store.EventsUpsert([]store.Event{e}); err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, nil, http.StatusCreated)
	}
}

const expiry = 3.0

// Get new event data from TBA only if event data is over 3 hours old.
// Upsert event data into database.
func (s *Server) updateEvents() error {
	now := time.Now()

	if s.eventsLastUpdate == nil || now.Sub(*s.eventsLastUpdate).Hours() > expiry {
		fullEvents, err := s.TBA.GetEvents(s.Year)
		if _, ok := err.(tba.ErrNotModified); ok {
			return nil
		} else if err != nil {
			return err
		}

		if err := s.Store.EventsUpsert(fullEvents); err != nil {
			return fmt.Errorf("upserting events: %v", err)
		}

		s.eventsLastUpdate = &now
	}

	return nil
}
