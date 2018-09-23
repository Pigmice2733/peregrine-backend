package store

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	// Register lib/pq PostreSQL driver
	_ "github.com/lib/pq"
)

// Service is an interface to manipulate the data store.
type Service struct {
	db *sql.DB
}

// Options holds information for connecting to a PostgreSQL instance.
type Options struct {
	User, Pass string
	Host       string
	Port       int
	DBName     string
	SSLMode    string
}

// ConnectionInfo returns the PostgreSQL connection string from an options struct.
func (o Options) ConnectionInfo() string {
	return fmt.Sprintf("host='%s' port='%d' user='%s' password='%s' dbname='%s' sslmode='%s'", o.Host, o.Port, o.User, o.Pass, o.DBName, o.SSLMode)
}

// New creates a new store service.
func New(o Options) (Service, error) {
	service := Service{}
	var err error

	service.db, err = sql.Open("postgres", o.ConnectionInfo())
	return service, err
}

// UnixTime exists so that we can have times that look like time.Time's to
// database drivers and JSON marshallers/unmarshallers but are internally
// represented as unix timestamps for easier comparison.
type UnixTime struct {
	unix int64
}

// NewUnix creates a new UnixTime timestamp from a time.Time.
func NewUnix(time time.Time) UnixTime {
	return UnixTime{
		unix: time.Unix(),
	}
}

// Scan accepts either a time.Time or an int64 for scanning from a database into
// a unix timestamp.
func (ut *UnixTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time:
		ut.unix = v.Unix()
	case int64:
		ut.unix = v
	default:
		return fmt.Errorf("got invalid type for time: %T", src)
	}

	return nil
}

// Value returns a driver.Value that is always a time.Time that represents the
// internally stored unix time.
func (ut *UnixTime) Value() (driver.Value, error) {
	return time.Unix(ut.unix, 0), nil
}

// MarshalJSON returns a []byte that represents this UnixTime in RFC 3339 format.
func (ut *UnixTime) MarshalJSON() ([]byte, error) {
	return time.Unix(ut.unix, 0).MarshalJSON()
}

// UnmarshalJSON accepts a []byte representing a time.Time value, and unmarshals
// it into a unix timestamp.
func (ut *UnixTime) UnmarshalJSON(data []byte) error {
	var time time.Time
	err := json.Unmarshal(data, &time)
	if err != nil {
		return err
	}
	ut.unix = time.Unix()
	return nil
}
