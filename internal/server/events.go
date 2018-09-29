package server

import (
	"net/http"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

type event struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	District  *string         `json:"district,omitempty"`
	Week      *int            `json:"week,omitempty"`
	StartDate *store.UnixTime `json:"startDate"`
	EndDate   *store.UnixTime `json:"endDate"`
	Location  struct {
		Name *string `json:"name,omitempty"`
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
	} `json:"location"`
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
		err := s.updateEvents()
		if err != nil {
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

		var events []event
		for _, fullEvent := range fullEvents {
			events = append(events, event{
				ID:        fullEvent.ID,
				Name:      fullEvent.Name,
				District:  fullEvent.District,
				Week:      fullEvent.Week,
				StartDate: &fullEvent.StartDate,
				EndDate:   &fullEvent.EndDate,
				Location: struct {
					Name *string `json:"name,omitempty"`
					Lat  float64 `json:"lat"`
					Lon  float64 `json:"lon"`
				}{
					Lat: fullEvent.Location.Lat,
					Lon: fullEvent.Location.Lon,
				},
			})
		}

		err = ihttp.Respond(w, events, nil, http.StatusOK)
		if err != nil {
			s.logger.Println(err)
		}
	}
}

// eventHandler returns a handler to get a specific event.
func (s *Server) eventHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get new event data from TBA if event data is over 24 hours old
		err := s.updateEvents()
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: updating event data: %v\n", err)
			return
		}

		vars := mux.Vars(r)
		eventKey := vars["eventKey"]

		fullEvent, err := s.store.GetEvent(eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.logger.Printf("Error: retrieving event data: %v\n", err)
			return
		}

		var webcasts []webcast
		for _, fullWebcast := range fullEvent.Webcasts {
			webcasts = append(webcasts, webcast{
				Type: string(fullWebcast.Type),
				URL:  fullWebcast.URL,
			})
		}

		event := webcastEvent{
			event: event{
				ID:        fullEvent.ID,
				Name:      fullEvent.Name,
				District:  fullEvent.District,
				Week:      fullEvent.Week,
				StartDate: &fullEvent.StartDate,
				EndDate:   &fullEvent.EndDate,
				Location: struct {
					Name *string `json:"name,omitempty"`
					Lat  float64 `json:"lat"`
					Lon  float64 `json:"lon"`
				}{
					Name: &fullEvent.Location.Name,
					Lat:  fullEvent.Location.Lat,
					Lon:  fullEvent.Location.Lon,
				},
			},
			Webcasts: webcasts,
		}

		err = ihttp.Respond(w, event, nil, http.StatusOK)
		if err != nil {
			s.logger.Println(err)
		}
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
		err = s.store.EventsUpsert(fullEvents)
		if err != nil {
			return err
		}
		s.eventsLastUpdate = &now
	}
	return nil
}
