package server

import (
	"net/http"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
)

type event struct {
	Name      string    `json:"name"`
	District  *string   `json:"district,omitempty"`
	Week      *int      `json:"week,omitempty"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
	Location  struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"location"`
}

// eventsHandler returns a handler to get all events in a given year
func (s *Server) eventsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fullEvents, err := s.tba.GetEvents(s.year)
		if err != nil {
			ihttp.ServerError(w)
			s.logger.Println(err)
			return
		}

		var events []event
		for _, fullEvent := range fullEvents {
			events = append(events, event{
				Name:      fullEvent.Name,
				District:  fullEvent.District,
				Week:      fullEvent.Week,
				StartDate: fullEvent.StartDate,
				EndDate:   fullEvent.EndDate,
				Location: struct {
					Lat float64 `json:"lat"`
					Lon float64 `json:"lon"`
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
