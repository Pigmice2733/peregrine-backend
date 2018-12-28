package store

import (
	"encoding/json"
)

// A Stat holds a single statistic from a single match, and could be either a
// boolean or numeric statistic
type Stat struct {
	// Numeric
	Attempts  *int `json:"attempts"`
	Successes *int `json:"successes"`
	// Boolean
	Attempted *bool `json:"attempted"`
	Succeeded *bool `json:"succeeded"`

	Name string `json:"statName"`
}

// ReportData holds all the data in a report
type ReportData struct {
	Auto   []Stat `json:"auto"`
	Teleop []Stat `json:"teleop"`
}

// Report is data about how an FRC team performed in a specific match.
type Report struct {
	ID       int64  `json:"-" db:"id"`
	MatchKey string `json:"-" db:"match_key"`
	TeamKey  string `json:"-" db:"team_key"`

	ReporterID *int64 `json:"reporterId" db:"reporter_id"`
	RealmID    *int64 `json:"-" db:"realm_id"`

	AutoName string          `json:"autoName" db:"auto_name"`
	Data     json.RawMessage `json:"data" db:"data"`
}

// UpsertReport creates a new report in the db, or replaces the existing one if
// the same reporter already has a report in the db for that team and match.
func (s *Service) UpsertReport(r Report) error {
	_, err := s.db.NamedExec(`
	INSERT
		INTO
			reports (match_key, team_key, reporter_id, realm_id, auto_name, data)
		VALUES (:match_key, :team_key, :reporter_id, :realm_id, :auto_name, :data)
		ON CONFLICT (match_key, team_key, reporter_id) DO
			UPDATE
				SET
					auto_name = :auto_name,
					data = :data
	`, r)
	return err
}

// GetTeamMatchReports retrieves all reports for a specific team and match from the db.
func (s *Service) GetTeamMatchReports(matchKey string, teamKey string) ([]Report, error) {
	reports := []Report{}

	return reports, s.db.Select(&reports, "SELECT * FROM reports WHERE match_key = $1 AND team_key = $2", matchKey, teamKey)
}

// GetEventReports retrieves all reports for an event from the db. If a realmID
// is specified, only reports from that realm will be included.
func (s *Service) GetEventReports(eventKey string, realmID *int64) ([]Report, error) {
	reports := []Report{}

	if realmID != nil {
		return reports, s.db.Select(&reports, `
	SELECT reports.* 
		FROM
			reports
		INNER JOIN
			matches m
		ON
			m.key = match_key
		WHERE
		    realm_id = $1 AND
			m.event_key = $2
	`, realmID, eventKey)
	}

	return reports, s.db.Select(&reports, `
	SELECT reports.* 
		FROM
			reports
		INNER JOIN
			matches m
		ON
			m.key = match_key
		WHERE
		    m.event_key = $1
	`, eventKey)
}

// GetTeamEventReports retrieves all reports for a specific team and event from
// the db. If a realmID is specified, only reports from that realm will be included.
func (s *Service) GetTeamEventReports(eventKey string, teamKey string, realmID *int64) ([]Report, error) {
	reports := []Report{}

	if realmID != nil {
		return reports, s.db.Select(&reports, `
	SELECT reports.* 
		FROM
			reports
		INNER JOIN
			matches m
		ON
			m.key = match_key
		WHERE
		    realm_id = $1 AND
			team_key = $2 AND
			m.event_key = $3
	`, realmID, teamKey, eventKey)
	}

	return reports, s.db.Select(&reports, `
	SELECT reports.* 
		FROM
			reports
		INNER JOIN
			matches m
		ON
			m.key = match_key
		WHERE
		    team_key = $1 AND
			m.event_key = $2
	`, teamKey, eventKey)
}

// GetReportsBySchemaID retrieves all reports with a specific schema.
func (s *Service) GetReportsBySchemaID(schemaID int64) ([]Report, error) {
	reports := []Report{}

	return reports, s.db.Select(&reports, `
	SELECT reports.*
	FROM reports, matches, events
	WHERE
		reports.match_key = matches.key
		AND matches.event_key = events.key
		AND event.schema_id = $1
	`, schemaID)
}
