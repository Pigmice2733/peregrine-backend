package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// ReportSection describes a single statistic in a schema
type ReportSection []json.RawMessage

// Value converts a ReportSection into JSON for PostgreSQL's JSONB type.
func (r *ReportSection) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// Scan converts data from PostgreSQL's JSONB type into ReportSection.
func (r *ReportSection) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("got incorrect type for JSONB")
	}

	return json.Unmarshal(bytes, r)
}

// Report is data about how an FRC team performed in a specific match.
type Report struct {
	ID       int64  `json:"-" db:"id"`
	MatchKey string `json:"-" db:"match_key"`
	TeamKey  string `json:"-" db:"team_key"`

	Reporter   string `json:"reporter" db:"reporter"`
	ReporterID *int64 `json:"reporterId" db:"reporter_id"`
	RealmID    *int64 `json:"-" db:"realm_id"`

	AutoName string         `json:"autoName" db:"auto_name"`
	Auto     *ReportSection `json:"auto" db:"auto"`
	Teleop   *ReportSection `json:"teleop" db:"teleop"`
}

// UpsertReport creates a new report in the db, or replaces the existing one if
// the same reporter already has a report in the db for that team and match.
func (s *Service) UpsertReport(r Report) error {
	_, err := s.db.NamedExec(`
	INSERT
		INTO
			reports (match_key, team_key, reporter, reporter_id, realm_id, auto_name, auto, teleop)
		VALUES (:match_key, :team_key, :reporter, :reporter_id, :realm_id, :auto_name, :auto, :teleop)
		ON CONFLICT (match_key, team_key, reporter) DO
			UPDATE
				SET
					auto_name = :auto_name,
					auto = :auto,
					teleop = :teleop
	`, r)
	return err
}

// GetReports retrieves all reports for a specific team and match from the db.
func (s *Service) GetReports(matchKey string, teamKey string) ([]Report, error) {
	reports := []Report{}

	return reports, s.db.Select(&reports, "SELECT * FROM reports WHERE match_key = $1 AND team_key = $2", matchKey, teamKey)
}
