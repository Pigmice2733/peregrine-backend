package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Register lib/pq PostreSQL driver
	"github.com/sirupsen/logrus"
)

// ErrNoResults indicates that no data matching the query was found.
type ErrNoResults struct {
	error
}

// ErrExists is returned if a unique record already exists.
type ErrExists struct {
	error
}

// ErrFKeyViolation is returned if inserting a record causes a foreign key violation.
type ErrFKeyViolation struct {
	error
}

const pgExists = "23505"
const pgFKeyViolation = "23503"

// Service provides methods for storing data in a PostgreSQL database.
type Service struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// New creates a new store service from a dataSourceName. The logger is used to
// log errors that would not otherwise be returned such as issues rolling back
// transactions. The context passed is used for pinging the database.
func New(ctx context.Context, dsn string, logger *logrus.Logger) (Service, error) {
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return Service{}, err
	}

	s := Service{db: db, logger: logger}
	return s, s.Ping(ctx)
}

// Ping pings the underlying postgresql database. You would think we would call
// db.Ping() here, but that doesn't actually Ping the database because reasons.
func (s *Service) Ping(ctx context.Context) error {
	if s.db != nil {
		return s.db.QueryRowContext(ctx, "SELECT 1").Scan(new(bool))
	}

	return errors.New("not connected to postgresql")
}

// Close closes the underlying postgresql database.
func (s *Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}

	return nil
}

func (s *Service) logErr(err error) {
	if err != nil {
		s.logger.Error(err)
	}
}

func (s *Service) doTransaction(ctx context.Context, txWrapper func(*sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	if err := txWrapper(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			s.logErr(errors.Wrap(err, "unable to rollback transaction"))
		}

		return errors.Wrap(err, "error in transaction wrapper")
	}

	return errors.Wrap(tx.Commit(), "unable to commit transaction")
}

// UnixTime exists so that we can have times that look like time.Time's to
// database drivers and JSON marshallers/unmarshallers but are internally
// represented as unix timestamps for easier comparison.
type UnixTime struct {
	Unix int64
}

// NewUnixFromTime creates a new UnixTime timestamp from a time.Time.
func NewUnixFromTime(time time.Time) UnixTime {
	return UnixTime{
		Unix: time.Unix(),
	}
}

// NewUnixFromInt creates a new UnixTime timestamp from an int64.
func NewUnixFromInt(time int64) UnixTime {
	return UnixTime{
		Unix: time,
	}
}

// Scan accepts either a time.Time or an int64 for scanning from a database into
// a unix timestamp.
func (ut *UnixTime) Scan(src interface{}) error {
	if ut == nil {
		return errors.New("cannot scan into nil unix time")
	}

	switch v := src.(type) {
	case time.Time:
		ut.Unix = v.Unix()
	case int64:
		ut.Unix = v
	default:
		return fmt.Errorf("got invalid type for time: %T", src)
	}

	return nil
}

// Value returns a driver.Value that is always a time.Time that represents the
// internally stored unix time.
func (ut UnixTime) Value() (driver.Value, error) {
	return time.Unix(ut.Unix, 0), nil
}

// MarshalJSON returns a []byte that represents this UnixTime in RFC 3339 format.
func (ut *UnixTime) MarshalJSON() ([]byte, error) {
	return time.Unix(ut.Unix, 0).MarshalJSON()
}

// UnmarshalJSON accepts a []byte representing a time.Time value, and unmarshals
// it into a unix timestamp.
func (ut *UnixTime) UnmarshalJSON(data []byte) error {
	var time time.Time

	if err := json.Unmarshal(data, &time); err != nil {
		return err
	}

	ut.Unix = time.Unix()

	return nil
}
