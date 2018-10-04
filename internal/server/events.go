package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

type location struct {
	Name *string `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type event struct {
	Key       string         `json:"key"`
	Name      string         `json:"name"`
	District  *string        `json:"district,omitempty"`
	Week      *int           `json:"week,omitempty"`
	StartDate store.UnixTime `json:"startDate"`
	EndDate   store.UnixTime `json:"endDate"`
	Location  location       `json:"location"`
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
			s.logger.Printf("Error: updating event data: %v\n", err)
			return
		}

		fullEvents, err := s.store.GetEvents()
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: retrieving event data: %v\n", err)
			return
		}

		events := []event{}
		for _, fullEvent := range fullEvents {
			events = append(events, event{
				Key:       fullEvent.Key,
				Name:      fullEvent.Name,
				District:  fullEvent.District,
				Week:      fullEvent.Week,
				StartDate: fullEvent.StartDate,
				EndDate:   fullEvent.EndDate,
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
			s.logger.Printf("Error: updating event data: %v\n", err)
			return
		}

		eventKey := mux.Vars(r)["eventKey"]

		fullEvent, err := s.store.GetEvent(eventKey)
		if err != nil {
			if store.IsNoResultError(err) {
				ihttp.Error(w, http.StatusNotFound)
				return
			}
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: retrieving event data: %v\n", err)
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
				Key:       fullEvent.Key,
				Name:      fullEvent.Name,
				District:  fullEvent.District,
				Week:      fullEvent.Week,
				StartDate: fullEvent.StartDate,
				EndDate:   fullEvent.EndDate,
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

// Get new event data from TBA only if event data is over 24 hours old.
// Upsert event data into database.
func (s *Server) updateEvents() error {
	now := time.Now()

	if s.eventsLastUpdate == nil || now.Sub(*s.eventsLastUpdate).Hours() > 24.0 {
		fullEvents, err := s.tba.GetEvents(s.year)
		if err != nil {
			return err
		}

		if err := s.store.EventsUpsert(fullEvents); err != nil {
			return fmt.Errorf("upserting events: %v", err)
		}

		s.eventsLastUpdate = &now
	}

	return nil
}
