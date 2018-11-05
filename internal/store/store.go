package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Register lib/pq PostreSQL driver
)

// ErrNoResults indicates that no data matching the query was found.
var ErrNoResults = fmt.Errorf("no matching records found")

// ErrExists is returned if a unique record already exists.
var ErrExists = fmt.Errorf("a record already exists")

// ErrFKeyViolation is returned if inserting a record causes a foreign key violation.
var ErrFKeyViolation = fmt.Errorf("foreign key violation")

const pgExists = "23505"
const pgFKeyViolation = "23503"

// Service is an interface to manipulate the data store.
type Service struct {
	db *sqlx.DB
}

// Options holds information for connecting to a PostgreSQL instance.
type Options struct {
	User    string `yaml:"user"`
	Pass    string `yaml:"pass"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Name    string `yaml:"name"`
	SSLMode string `yaml:"sslMode"`
}

// ConnectionInfo returns the PostgreSQL connection string from an options struct.
func (o Options) ConnectionInfo() string {
	return fmt.Sprintf("host='%s' port='%d' user='%s' password='%s' dbname='%s' sslmode='%s'", o.Host, o.Port, o.User, o.Pass, o.Name, o.SSLMode)
}

// New creates a new store service.
func New(o Options) (Service, error) {
	db, err := sqlx.Open("postgres", o.ConnectionInfo())
	if err != nil {
		return Service{}, err
	}

	return Service{db: db}, db.Ping()
}

// Ping pings the underlying postgresql database. You would think we would call
// db.Ping() here, but that doesn't actually Ping the database because reasons.
func (s *Service) Ping() error {
	if s.db != nil {
		var alive bool
		err := s.db.QueryRow("SELECT 1").Scan(&alive)
		return err
	}

	return fmt.Errorf("not connected to postgresql")
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
		return fmt.Errorf("cannot scan into nil unix time")
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
