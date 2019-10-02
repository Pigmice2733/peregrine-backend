package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
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
	StartDate    time.Time      `json:"startDate" db:"start_date"`
	EndDate      time.Time      `json:"endDate" db:"end_date"`
	Webcasts     pq.StringArray `json:"webcasts" db:"webcasts"`
	LocationName string         `json:"locationName" db:"location_name"`
	GMapsURL     *string        `json:"gmapsUrl" db:"gmaps_url"`
	Lat          float64        `json:"lat" db:"lat"`
	Lon          float64        `json:"lon" db:"lon"`
	TBADeleted   bool           `json:"tbaDeleted" db:"tba_deleted"`
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
	gmaps_url,
	lat,
	lon,
	tba_deleted,
	events.realm_id,
	COALESCE(schema_id, s.id) AS schema_id
FROM
	events
LEFT JOIN
	schemas s
ON
	s.year = EXTRACT(YEAR FROM start_date)`

// GetEvents returns all events from the database. event.Webcasts and schemaID will be nil for every event.
// If tbaDeleted is true, events that have been deleted from TBA will be returned in addition to events that
// have not been deleted. Otherwise, only events that have not been deleted will be returned.
func (s *Service) GetEvents(ctx context.Context, tbaDeleted bool) (events []Event, err error) {
	query := eventsQuery

	if !tbaDeleted {
		query += " WHERE NOT tba_deleted"
	}

	events = make([]Event, 0)
	return events, s.db.SelectContext(ctx, &events, query)
}

const eventsRealmQuery = eventsQuery + `WHERE (events.realm_id IS NULL OR events.realm_id = $1)`

// GetEventsForRealm returns all events from a specific realm. Additionally all
// TBA events will be retrieved. If no realm is specified (nil) then just the TBA
// events will be retrieved. event.Webcasts and schemaID will be nil for every event.
// If tbaDeleted is true, events that have been deleted from TBA will be returned in addition to events that
// have not been deleted. Otherwise, only events that have not been deleted will be returned.
func (s *Service) GetEventsForRealm(ctx context.Context, tbaDeleted bool, realmID *int64) (events []Event, err error) {
	query := eventsRealmQuery

	if !tbaDeleted {
		query += " AND NOT tba_deleted"
	}

	events = make([]Event, 0)
	return events, s.db.SelectContext(ctx, &events, query, realmID)
}

// GetEventForRealm retrieves a specific event in a specific realm (or no realm for TBA events).
func (s *Service) GetEventForRealm(ctx context.Context, eventKey string, realmID *int64) (event Event, err error) {
	err = s.db.GetContext(ctx, &event, eventsRealmQuery+" AND key = $2", realmID, eventKey)
	if err == sql.ErrNoRows {
		return event, ErrNoResults{errors.Wrapf(err, "event %s does not exist", eventKey)}
	}

	return event, err
}

// GetActiveEvents returns all events that are currently happening. If tbaDeleted is true,
// events that have been deleted from TBA will be returned in addition to events that have
// not been deleted. Otherwise, only events that have not been deleted will be returned.
func (s *Service) GetActiveEvents(ctx context.Context, tbaDeleted bool) ([]Event, error) {
	query := `
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
		tba_deleted,
		events.realm_id,
		COALESCE(schema_id, s.id) AS schema_id
	FROM
		events
	LEFT JOIN
		schemas s
	ON
		s.year = EXTRACT(YEAR FROM start_date)
	WHERE
		start_date <= CURRENT_DATE
		AND end_date >= CURRENT_DATE`

	if !tbaDeleted {
		query += " AND NOT tba_deleted"
	}

	events := []Event{}
	return events, s.db.SelectContext(ctx, &events, query)
}

// EventsUpsert upserts multiple events into the database. It will set tba_deleted
// to false for all updated events. schema_id will only be updated if null.
func (s *Service) EventsUpsert(ctx context.Context, events []Event) error {
	return s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		eventStmt, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO events (key, name, district, full_district, week, start_date, end_date, webcasts, location_name, gmaps_url, lat, lon, realm_id, schema_id, tba_deleted)
		VALUES (:key, :name, :district, :full_district, :week, :start_date, :end_date, :webcasts, :location_name, :gmaps_url, :lat, :lon, :realm_id, :schema_id, :tba_deleted)
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
					gmaps_url = :gmaps_url,
					lat = :lat,
					lon = :lon,
					realm_id = :realm_id,
					schema_id = COALESCE(events.schema_id, :schema_id),
					tba_deleted = false
		`)
		if err != nil {
			return errors.Wrap(err, "unable to prepare events upsert statemant")
		}
		defer eventStmt.Close()

		for _, event := range events {
			if _, err = eventStmt.ExecContext(ctx, event); err != nil {
				if err, ok := err.(*pq.Error); ok {
					if err.Code == pgExists {
						return ErrExists{errors.Wrapf(err, "event with key %s already exists", event.Key)}
					} else if err.Code == pgFKeyViolation {
						return ErrFKeyViolation{errors.Wrap(err, "foreign key violation")}
					}
				}
				return errors.Wrap(err, "unable to upsert events")
			}
		}

		return nil
	})
}

// MarkEventsDeleted will set tba_deleted to true on all events that were
// *not* included in the events slice and are not custom events (have a NULL realm_id).
func (s *Service) MarkEventsDeleted(ctx context.Context, events []Event) error {
	keys := pq.StringArray{}
	for _, e := range events {
		keys = append(keys, e.Key)
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE events
			SET
				tba_deleted = true
			WHERE
				key != ALL($1) AND
				realm_id IS NULL
	`, keys)

	return errors.Wrap(err, "unable to mark tba_deleted on missing events")
}

// ExclusiveLockEventsTx acquires an exclusive lock on the events table.
func (s *Service) ExclusiveLockEventsTx(ctx context.Context, tx *sqlx.Tx) error {
	_, err := tx.Exec("LOCK TABLE events IN EXCLUSIVE MODE")
	return errors.Wrap(err, "unable to lock events")
}

// GetEventRealmIDTx returns the realm ID of an event by key.
func (s *Service) GetEventRealmIDTx(ctx context.Context, tx *sqlx.Tx, eventKey string) (realmID *int64, err error) {
	err = tx.QueryRowContext(ctx, "SELECT realm_id FROM events WHERE key = $1", eventKey).Scan(&realmID)
	if err == sql.ErrNoRows {
		return nil, ErrNoResults{errors.Wrap(err, "couldn't find event by key")}
	}

	return realmID, errors.Wrap(err, "unable to determine event realm ID")
}

// UpsertEventTx upserts a single event into the database and returns whether
// the event was created or updated.
func (s *Service) UpsertEventTx(ctx context.Context, tx *sqlx.Tx, event Event) error {
	_, err := tx.NamedExecContext(ctx, `
			INSERT INTO events (key, name, district, full_district, week, start_date, end_date, webcasts, location_name, gmaps_url, lat, lon, realm_id, schema_id, tba_deleted)
				VALUES (:key, :name, :district, :full_district, :week, :start_date, :end_date, :webcasts, :location_name, :gmaps_url, :lat, :lon, :realm_id, :schema_id, :tba_deleted)
			ON CONFLICT (key) DO
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
						gmaps_url= :gmaps_url,
						lat = :lat,
						lon = :lon,
						realm_id = :realm_id,
						schema_id = :schema_id,
						tba_deleted = :tba_deleted
		`, event)

	return errors.Wrap(err, "unable to upsert event")
}
