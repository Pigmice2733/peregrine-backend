package store

import (
	"database/sql"
)

// Event holds information about an FRC event such as webcast associated with
// it, the location, it's start date, and more.
type Event struct {
	Key       string
	Name      string
	District  *string
	Week      *int
	StartDate UnixTime
	EndDate   UnixTime
	Webcasts  []Webcast
	Location  Location
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
	rows, err := s.db.Query("SELECT key, name, district, week, start_date, end_date, location_name, lat, lon FROM events")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event Event
		event.Location = Location{}
		if err := rows.Scan(&event.Key, &event.Name, &event.District, &event.Week, &event.StartDate, &event.EndDate, &event.Location.Name, &event.Location.Lat, &event.Location.Lon); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetEvent retrieves a specific event.
func (s *Service) GetEvent(eventKey string) (Event, error) {
	event := Event{Key: eventKey, Location: Location{}}
	if err := s.db.QueryRow("SELECT name, district, week, start_date, end_date, location_name, lat, lon FROM events WHERE key = $1", eventKey).
		Scan(&event.Name, &event.District, &event.Week, &event.StartDate, &event.EndDate, &event.Location.Name, &event.Location.Lat, &event.Location.Lon); err != nil {
		if err == sql.ErrNoRows {
			return event, NoResultError{err}
		}
		return event, err
	}

	rows, err := s.db.Query("SELECT type, url FROM webcasts WHERE event_key = $1", eventKey)
	if err != nil {
		return event, err
	}
	defer rows.Close()

	for rows.Next() {
		var webcast Webcast
		var webcastType string
		if err := rows.Scan(&webcastType, &webcast.URL); err != nil {
			return event, err
		}
		webcast.Type = WebcastType(webcastType)
		event.Webcasts = append(event.Webcasts, webcast)
	}

	return event, rows.Err()
}

// EventsUpsert upserts multiple events into the database.
func (s *Service) EventsUpsert(events []Event) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	eventStmt, err := tx.Prepare(`
		INSERT INTO events (key, name, district, week, start_date, end_date, location_name, lat, lon)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (key)
		DO
			UPDATE
				SET name = $2, district = $3, week = $4, start_date = $5, end_date = $6, location_name = $7, lat = $8, lon = $9
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer eventStmt.Close()

	deleteWebcastsStmt, err := tx.Prepare(`
	    DELETE FROM webcasts WHERE event_key = $1
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer deleteWebcastsStmt.Close()

	webcastStmt, err := tx.Prepare(`
		INSERT INTO webcasts (event_key, type, url)
		VALUES ($1, $2, $3)
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer webcastStmt.Close()

	for _, event := range events {
		if _, err = eventStmt.Exec(event.Key, event.Name, event.District, event.Week, &event.StartDate, &event.EndDate,
			event.Location.Name, event.Location.Lat, event.Location.Lon); err != nil {
			_ = tx.Rollback()
			return err
		}

		if _, err = deleteWebcastsStmt.Exec(event.Key); err != nil {
			_ = tx.Rollback()
			return err
		}

		for _, webcast := range event.Webcasts {
			if _, err = webcastStmt.Exec(event.Key, webcast.Type, webcast.URL); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}
