package store

// Event holds information about an FRC event such as webcast associated with
// it, the location, it's start date, and more.
type Event struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	District  *string   `json:"district,omitempty"`
	Week      *int      `json:"week,omitempty"`
	StartDate UnixTime  `json:"startDate"`
	EndDate   UnixTime  `json:"endDate"`
	Webcasts  []Webcast `json:"webcasts,omitempty"`
	Location  Location  `json:"location"`
}

// WebcastType represents a data source for a webcast such as twitch or youtube.
type WebcastType string

const (
	// Twitch provides livestreams of events.
	Twitch WebcastType = "twitch"
	// Youtube provides livestreams of events.
	Youtube WebcastType = "youtube"
)

// Webcast represents a webcast of an events.
type Webcast struct {
	Type WebcastType
	URL  string
}

// Location holds a location for events: a name, and a latlong.
type Location struct {
	Name string
	Lat  float64
	Lon  float64
}

// GetEvents returns all events from the database. event.Webcasts will be nil for every event.
func (s *Service) GetEvents() ([]Event, error) {
	var events []Event

	rows, err := s.db.Query("SELECT id, name, district, week, startDate, endDate, locationName, lat, lon FROM events")
	if err != nil {
		return events, err
	}
	defer rows.Close()

	for rows.Next() {
		var event Event
		event.Location = Location{}
		if err := rows.Scan(&event.ID, &event.Name, &event.District, &event.Week, &event.StartDate, &event.EndDate, &event.Location.Name, &event.Location.Lat, &event.Location.Lon); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// EventsUpsert upserts multiple events into the database.
func (s *Service) EventsUpsert(events []Event) error {
	eventStmt, err := s.db.Prepare(`
		INSERT INTO events (id, name, district, week, startDate, endDate, locationName, lat, lon)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id)
		DO
			UPDATE
				SET name = $2, district = $3, week = $4, startDate = $5, endDate = $6, locationName = $7, lat = $8, lon = $9
	`)
	if err != nil {
		return err
	}
	defer eventStmt.Close()

	deleteWebcastsStmt, err := s.db.Prepare(`
	    DELETE FROM webcasts WHERE eventID = $1
	`)
	if err != nil {
		return err
	}
	defer deleteWebcastsStmt.Close()

	webcastStmt, err := s.db.Prepare(`
		INSERT INTO webcasts (eventID, type, url)
		VALUES ($1, $2, $3)
	`)
	if err != nil {
		return err
	}
	defer webcastStmt.Close()

	for _, event := range events {
		if _, err := eventStmt.Exec(event.ID, event.Name, event.District, event.Week, &event.StartDate, &event.EndDate,
			event.Location.Name, event.Location.Lat, event.Location.Lon); err != nil {
			return err
		}

		if _, err := deleteWebcastsStmt.Exec(event.ID); err != nil {
			return err
		}

		for _, webcast := range event.Webcasts {
			if _, err := webcastStmt.Exec(event.ID, webcast.Type, webcast.URL); err != nil {
				return err
			}
		}
	}

	return nil
}
