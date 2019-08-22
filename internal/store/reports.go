package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

// A Stat holds a single statistic from a single match, and could be either a
// boolean or numeric statistic
type Stat struct {
	Value float64 `json:"value"`
	Name  string  `json:"name"`
}

// ReportData holds all the data in a report
type ReportData []Stat

// Value implements driver.Valuer to return JSON for the DB from ReportData.
func (rd ReportData) Value() (driver.Value, error) { return json.Marshal(rd) }

// Scan implements sql.Scanner to scan JSON from the DB into ReportData.
func (rd *ReportData) Scan(src interface{}) error {
	j, ok := src.([]byte)
	if !ok {
		return errors.New("got invalid type for ReportData")
	}

	return json.Unmarshal(j, rd)
}

// Report is data about how an FRC team performed in a specific match.
type Report struct {
	ID         int64      `json:"-" db:"id"`
	MatchKey   string     `json:"-" db:"match_key"`
	TeamKey    string     `json:"-" db:"team_key"`
	ReporterID *int64     `json:"reporterId" db:"reporter_id"`
	RealmID    *int64     `json:"-" db:"realm_id"`
	Data       ReportData `json:"data" db:"data"`
}

// Leaderboard holds information about how many reports each reporter submitted.
type Leaderboard []struct {
	ReporterID int64 `json:"reporterId" db:"reporter_id"`
	Reports    int64 `json:"reports" db:"num_reports"`
}

// UpsertReport creates a new report in the db, or replaces the existing one if
// the same reporter already has a report in the db for that team and match. It
// returns a boolean that is true when the report was created, and false when it
// was updated.
func (s *Service) UpsertReport(ctx context.Context, r Report) (created bool, err error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, errors.Wrap(err, "unable to begin transaction for report upsert")
	}

	if _, err := tx.Exec("LOCK TABLE reports IN EXCLUSIVE MODE"); err != nil {
		s.logErr(errors.Wrap(tx.Rollback(), "rolling back report upsert tx"))
		return false, errors.Wrap(err, "unable to lock reports")
	}

	var existed bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT FROM reports
				WHERE match_key = $1 AND
				team_key = $2
		)
		`, r.MatchKey, r.TeamKey).Scan(&existed)
	if err != nil {
		s.logErr(errors.Wrap(tx.Rollback(), "rolling back report upsert tx"))
		return false, errors.Wrap(err, "unable to determine if report exists")
	}

	_, err = tx.NamedExecContext(ctx, `
	INSERT
		INTO
			reports (match_key, team_key, reporter_id, realm_id, data)
		VALUES (:match_key, :team_key, :reporter_id, :realm_id, :data)
		ON CONFLICT (match_key, team_key, reporter_id) DO
			UPDATE
				SET
					data = :data
	`, r)
	if err != nil {
		s.logErr(errors.Wrap(tx.Rollback(), "rolling back report upsert tx"))
		return false, errors.Wrap(err, "unable to upsert report")
	}

	return !existed, errors.Wrap(tx.Commit(), "unable to commit transaction")
}

// GetTeamMatchReports retrieves all reports for a specific team and match from the db.
func (s *Service) GetTeamMatchReports(ctx context.Context, matchKey string, teamKey string) ([]Report, error) {
	reports := []Report{}

	return reports, s.db.SelectContext(ctx, &reports, "SELECT * FROM reports WHERE match_key = $1 AND team_key = $2", matchKey, teamKey)
}

// GetEventReports retrieves all reports for an event from the db. If a realmID
// is specified, only reports from that realm will be included.
func (s *Service) GetEventReports(ctx context.Context, eventKey string, realmID *int64) ([]Report, error) {
	reports := []Report{}

	if realmID != nil {
		return reports, s.db.SelectContext(ctx, &reports, `
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

	return reports, s.db.SelectContext(ctx, &reports, `
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
func (s *Service) GetTeamEventReports(ctx context.Context, eventKey string, teamKey string, realmID *int64) ([]Report, error) {
	reports := []Report{}

	if realmID != nil {
		return reports, s.db.SelectContext(ctx, &reports, `
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

	return reports, s.db.SelectContext(ctx, &reports, `
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
func (s *Service) GetReportsBySchemaID(ctx context.Context, schemaID int64) ([]Report, error) {
	reports := []Report{}

	return reports, s.db.SelectContext(ctx, &reports, `
	SELECT reports.*
	FROM reports, matches, events
	WHERE
		reports.match_key = matches.key
		AND matches.event_key = events.key
		AND event.schema_id = $1
	`, schemaID)
}

// GetLeaderboard retrieves leaderboard information from the reports and users table.
func (s *Service) GetLeaderboard(ctx context.Context) (Leaderboard, error) {
	leaderboard := Leaderboard{}

	return leaderboard, s.db.SelectContext(ctx, &leaderboard, `
	SELECT
		users.id AS reporter_id, COUNT(reports.reporter_id) AS num_reports
	FROM users
	LEFT JOIN reports
		ON (users.id = reports.reporter_id)
	GROUP BY users.id
	ORDER BY num_reports DESC;
	`)
}
