package store

import (
	"database/sql"
	"fmt"
)

// Event holds information about an FRC event such as webcast associated with
// it, the location, its start date, and more.
type Event struct {
	Key           string    `json:"key" db:"key"`
	Name          string    `json:"name" db:"name"`
	District      *string   `json:"district" db:"district"`
	FullDistrict  *string   `json:"fullDistrict" db:"full_district"`
	Week          *int      `json:"week" db:"week"`
	ManuallyAdded bool      `json:"manuallyAdded" db:"manually_added"`
	StartDate     UnixTime  `json:"startDate" db:"start_date"`
	EndDate       UnixTime  `json:"endDate" db:"end_date"`
	Webcasts      []Webcast `json:"webcasts"`
	Location      `json:"location"`
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
	EventKey string      `json:"-" db:"event_key"`
	Type     WebcastType `json:"type" db:"type"`
	URL      string      `json:"url" db:"url"`
}

// Location holds a location for events: a name, and a latlong.
type Location struct {
	Name string  `json:"name" db:"location_name"`
	Lat  float64 `json:"lat" db:"lat"`
	Lon  float64 `json:"lon" db:"lon"`
}

// GetEvents returns all events from the database. event.Webcasts will be nil for every event.
func (s *Service) GetEvents() ([]Event, error) {
	events := []Event{}

	return events, s.db.Select(&events, "SELECT * FROM events")
}

// ErrManuallyAdded is returned when an event has been manually inserted into
// the DB rather than returned from TBA.
var ErrManuallyAdded = fmt.Errorf("store: manually added event")

// CheckTBAEventKeyExists checks whether a specific event key exists and is from
// TBA rather than manually added.
func (s *Service) CheckTBAEventKeyExists(eventKey string) error {
	var manuallyAdded bool

	err := s.db.Get(&manuallyAdded, "SELECT manually_added FROM events WHERE key = $1", eventKey)
	if err == sql.ErrNoRows {
		return ErrNoResults(fmt.Errorf("event key %s does not exist", eventKey))
	} else if err != nil {
		return err
	}

	if manuallyAdded {
		return ErrManuallyAdded
	}

	return nil
}

// GetEvent retrieves a specific event.
func (s *Service) GetEvent(eventKey string) (Event, error) {
	var event Event
	if err := s.db.Get(&event, "SELECT * FROM events WHERE key = $1", eventKey); err != nil {
		if err == sql.ErrNoRows {
			return event, ErrNoResults(fmt.Errorf("event %+v does not exist", event))
		}
		return event, err
	}

	err := s.db.Select(&event.Webcasts, "SELECT type, url FROM webcasts WHERE event_key = $1", eventKey)
	return event, err
}

// EventsUpsert upserts multiple events into the database.
func (s *Service) EventsUpsert(events []Event) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	eventStmt, err := tx.PrepareNamed(`
		INSERT INTO events (key, name, district, full_district, week, manually_added, start_date, end_date, location_name, lat, lon)
		VALUES (:key, :name, :district, :full_district, :week, :manually_added, :start_date, :end_date, :location_name, :lat, :lon)
		ON CONFLICT (key)
		DO
			UPDATE
				SET
					name = :name,
					district = :district,
					full_district = :full_district,
					week = :week,
					manually_added = :manually_added,
					start_date = :start_date,
					end_date = :end_date,
					location_name = :location_name,
					lat = :lat,
					lon = :lon
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

	webcastStmt, err := tx.PrepareNamed(`
		INSERT INTO webcasts (event_key, type, url)
		VALUES (:event_key, :type, :url)
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer webcastStmt.Close()

	for _, event := range events {
		if _, err = eventStmt.Exec(event); err != nil {
			_ = tx.Rollback()
			return err
		}

		if _, err = deleteWebcastsStmt.Exec(event.Key); err != nil {
			_ = tx.Rollback()
			return err
		}

		for _, webcast := range event.Webcasts {
			webcast.EventKey = event.Key
			if _, err = webcastStmt.Exec(webcast); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}
