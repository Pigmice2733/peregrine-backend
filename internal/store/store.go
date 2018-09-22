package store

import (
	"database/sql/driver"
	"fmt"
	"time"
)

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

// UnixTime exists so that we can have times that look like time.Time's to
// database drivers and JSON marshallers/unmarshallers but are internally
// represented as unix timestamps for easier comparison.
type UnixTime struct {
	unix int64
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

// Event holds information about an FRC event such as webcast associated with
// it, the location, it's start date, and more.
type Event struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	District  *string   `json:"district,omitempty"`
	Week      *int      `json:"week,omitempty"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
	Webcasts  []Webcast `json:"webcasts,omitempty"`
	Location  Location  `json:"location"`
}
