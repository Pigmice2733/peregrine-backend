package store

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Event holds information about an FRC event such as webcast associated with
// it, the location, its start date, and more.
type Event struct {
	Key          string    `json:"key" db:"key"`
	RealmID      *int64    `db:"realm_id"`
	Name         string    `json:"name" db:"name"`
	District     *string   `json:"district" db:"district"`
	FullDistrict *string   `json:"fullDistrict" db:"full_district"`
	Week         *int      `json:"week" db:"week"`
	StartDate    UnixTime  `json:"startDate" db:"start_date"`
	EndDate      UnixTime  `json:"endDate" db:"end_date"`
	Webcasts     []Webcast `json:"webcasts"`
	Location     `json:"location"`
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

// GetEventsFromRealm returns all events from a specific realm. Additionally all
// TBA events will be retrieved. If no realm is specified (nil) then just the TBA
// events will be retrieved. event.Webcasts will be nil for every event.
func (s *Service) GetEventsFromRealm(realm *int64) ([]Event, error) {
	events := []Event{}

	if realm == nil {
		return events, s.db.Select(&events, "SELECT * FROM events WHERE realm_id IS NULL")
	}
	return events, s.db.Select(&events, "SELECT * FROM events WHERE realm_id IS NULL OR realm_id = $1", *realm)
}

// CheckTBAEventKeyExists checks whether a specific event key exists and is from
// TBA rather than manually added.
func (s *Service) CheckTBAEventKeyExists(eventKey string) (bool, error) {
	var realmID *int64

	err := s.db.Get(&realmID, "SELECT realm_id FROM events WHERE key = $1", eventKey)
	if err == sql.ErrNoRows {
		return false, ErrNoResults{errors.Wrapf(err, "event key %s not found", eventKey)}
	} else if err != nil {
		return false, err
	}

	return realmID == nil, nil
}

// GetEvent retrieves a specific event.
func (s *Service) GetEvent(eventKey string) (Event, error) {
	var event Event
	if err := s.db.Get(&event, "SELECT * FROM events WHERE key = $1", eventKey); err != nil {
		if err == sql.ErrNoRows {
			return event, ErrNoResults{errors.Wrapf(err, "event %s does not exist", event.Key)}
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
		INSERT INTO events (key, name, district, full_district, week, start_date, end_date, location_name, lat, lon, realm_id)
		VALUES (:key, :name, :district, :full_district, :week, :start_date, :end_date, :location_name, :lat, :lon, :realm_id)
		ON CONFLICT (key)
		DO
			UPDATE
				SET
					name = :name,
					district = :district,
					full_district = :full_district,
					week = :week,
					start_date = :start_date,
					end_date = :end_date,
					location_name = :location_name,
					lat = :lat,
					lon = :lon,
                    realm_id = :realm_id
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
			if err, ok := err.(*pq.Error); ok {
				if err.Code == pgExists {
					return ErrExists{errors.Wrapf(err, "event with key %s already exists", event.Key)}
				} else if err.Code == pgFKeyViolation {
					return ErrFKeyViolation{errors.Wrap(err, "foreign key violation")}
				}
			}
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
