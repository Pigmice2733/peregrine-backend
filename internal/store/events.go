package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Event holds information about an FRC event such as webcast associated with
// it, the location, its start date, and more.
type Event struct {
	Key          string         `json:"key" db:"key"`
	RealmID      *int64         `json:"realmId,omitempty" db:"realm_id"`
	SchemaID     *int64         `json:"schemaId,omitempty" db:"schema_id"`
	Name         string         `json:"name" db:"name"`
	District     *string        `json:"district,omitempty" db:"district"`
	FullDistrict *string        `json:"fullDistrict,omitempty" db:"full_district"`
	Week         *int           `json:"week,omitempty" db:"week"`
	StartDate    UnixTime       `json:"startDate" db:"start_date"`
	EndDate      UnixTime       `json:"endDate" db:"end_date"`
	Webcasts     pq.StringArray `json:"webcasts" db:"webcasts"`
	LocationName string         `json:"locationName" db:"location_name"`
	Lat          float64        `json:"lat" db:"lat"`
	Lon          float64        `json:"lon" db:"lon"`
}

func abs(a int64) int64 {
	if a < 0 {
		return a * -1
	}

	return a
}

type Events []Event

func (e Events) Len() int      { return len(e) }
func (e Events) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e Events) Less(i, j int) bool {
	now := time.Now().Unix()

	iDiff := now - e[i].EndDate.Unix
	jDiff := now - e[j].EndDate.Unix

	if iDiff < 0 && jDiff >= 0 {
		return true
	} else if jDiff < 0 && iDiff >= 0 {
		return false
	}

	return abs(now-e[i].StartDate.Unix) < abs(now-e[j].StartDate.Unix)
}

const eventsQuery = `
SELECT
	    key,
		name,
		district,
		full_district,
		week,
		start_date,
		end_date,
		webcasts,
		location_name,
		lat,
		lon,
		events.realm_id,
		COALESCE(schema_id, s.id) AS schema_id
	FROM
		events
	LEFT JOIN
		schemas s
	ON
	    s.year = EXTRACT(YEAR FROM start_date)`

// GetEvents returns all events from the database. event.Webcasts and schemaID will be nil for every event.
func (s *Service) GetEvents(ctx context.Context) ([]Event, error) {
	events := []Event{}
	return events, s.db.SelectContext(ctx, &events, eventsQuery)
}

// GetEventsFromRealm returns all events from a specific realm. Additionally all
// TBA events will be retrieved. If no realm is specified (nil) then just the TBA
// events will be retrieved. event.Webcasts and schemaID will be nil for every event.
func (s *Service) GetEventsFromRealm(ctx context.Context, realm *int64) ([]Event, error) {
	events := []Event{}
	if realm == nil {
		return events, s.db.SelectContext(ctx, &events, eventsQuery+" WHERE events.realm_id IS NULL")
	}
	return events, s.db.SelectContext(ctx, &events, eventsQuery+" WHERE events.realm_id IS NULL OR events.realm_id = $1", *realm)
}

// CheckTBAEventKeyExists checks whether a specific event key exists and is from
// TBA rather than manually added. Returns ErrNoResults if event does not exist.
func (s *Service) CheckTBAEventKeyExists(ctx context.Context, eventKey string) (bool, error) {
	var realmID *int64

	err := s.db.GetContext(ctx, &realmID, "SELECT realm_id FROM events WHERE key = $1", eventKey)
	if err == sql.ErrNoRows {
		return false, ErrNoResults{errors.Wrapf(err, "event key %s not found", eventKey)}
	} else if err != nil {
		return false, err
	}

	return realmID == nil, nil
}

// GetEvent retrieves a specific event.
func (s *Service) GetEvent(ctx context.Context, eventKey string) (Event, error) {
	var event Event

	err := s.db.GetContext(ctx, &event, eventsQuery+" WHERE key = $1", eventKey)

	if err == sql.ErrNoRows {
		return event, ErrNoResults{errors.Wrapf(err, "event %s does not exist", event.Key)}
	}
	return event, err
}

// EventsUpsert upserts multiple events into the database.
func (s *Service) EventsUpsert(ctx context.Context, events []Event) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	eventStmt, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO events (key, name, district, full_district, week, start_date, end_date, webcasts, location_name, lat, lon, realm_id, schema_id)
		VALUES (:key, :name, :district, :full_district, :week, :start_date, :end_date, :webcasts, :location_name, :lat, :lon, :realm_id, :schema_id)
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
					webcasts = :webcasts,
					location_name = :location_name,
					lat = :lat,
					lon = :lon,
					realm_id = :realm_id,
					schema_id = :schema_id
	`)
	if err != nil {
		s.logErr(errors.Wrap(tx.Rollback(), "rolling back events upsert tx"))
		return err
	}
	defer eventStmt.Close()

	for _, event := range events {
		if _, err = eventStmt.ExecContext(ctx, event); err != nil {
			s.logErr(errors.Wrap(tx.Rollback(), "rolling back events upsert tx"))
			if err, ok := err.(*pq.Error); ok {
				if err.Code == pgExists {
					return ErrExists{errors.Wrapf(err, "event with key %s already exists", event.Key)}
				} else if err.Code == pgFKeyViolation {
					return ErrFKeyViolation{errors.Wrap(err, "foreign key violation")}
				}
			}
			return err
		}
	}

	return tx.Commit()
}

// UpsertEvent upserts a single event into the database and returns whether
// the event was created or updated.
func (s *Service) UpsertEvent(ctx context.Context, event Event) (created bool, err error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, errors.Wrap(err, "unable to begin transaction for event upsert")
	}

	if _, err := tx.Exec("LOCK TABLE events IN EXCLUSIVE MODE"); err != nil {
		s.logErr(errors.Wrap(tx.Rollback(), "rolling back event upsert tx"))
		return false, errors.Wrap(err, "unable to lock events")
	}

	var existed bool
	err = tx.QueryRow("SELECT EXISTS(SELECT FROM events WHERE key = $1)", event.Key).Scan(&existed)
	if err != nil {
		s.logErr(errors.Wrap(tx.Rollback(), "rolling back event upsert tx"))
		return false, errors.Wrap(err, "unable to determine if event exists")
	}

	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO events (key, name, district, full_district, week, start_date, end_date, webcasts, location_name, lat, lon, realm_id, schema_id)
		VALUES (:key, :name, :district, :full_district, :week, :start_date, :end_date, :webcasts, :location_name, :lat, :lon, :realm_id, :schema_id)
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
					webcasts = :webcasts,
					location_name = :location_name,
					lat = :lat,
					lon = :lon,
					realm_id = :realm_id,
					schema_id = :schema_id
	`, event)
	if err != nil {
		s.logErr(errors.Wrap(tx.Rollback(), "rolling back event upsert tx"))
		return false, errors.Wrap(err, "unable to upsert event")
	}

	return !existed, errors.Wrap(tx.Commit(), "unable to commit transaction")

}
