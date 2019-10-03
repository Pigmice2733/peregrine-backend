package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
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
	EventKey   string     `json:"-" db:"event_key"`
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
	var existed bool

	err = s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		var existed bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT FROM reports
				WHERE
					event_key = $1 AND
					match_key = $2 AND
					team_key = $3 AND
					reporter_id = $4
			)
			`, r.EventKey, r.MatchKey, r.TeamKey, r.ReporterID).Scan(&existed)
		if err != nil {
			return fmt.Errorf("unable to determine if report exists: %w", err)
		}

		_, err = tx.NamedExecContext(ctx, `
			INSERT INTO
				reports (event_key, match_key, team_key, reporter_id, realm_id, data)
			VALUES (:event_key, :match_key, :team_key, :reporter_id, :realm_id, :data)
			ON CONFLICT (event_key, match_key, team_key, reporter_id)
				DO UPDATE SET data = :data, realm_id = :realm_id
		`, r)
		if err != nil {
			return fmt.Errorf("unable to upsert report: %w", err)
		}

		return nil
	})

	return !existed, err
}

// GetEventReportsForRealm returns all event reports for a specific event and realm.
func (s *Service) GetEventReportsForRealm(ctx context.Context, eventKey string, realmID *int64) ([]Report, error) {
	const query = `
	SELECT reports.*
	FROM reports
	INNER JOIN realms
		ON realms.id = reports.realm_id
	WHERE
		reports.event_key = $1 AND
		(realms.share_reports = true OR realms.id = $2)`

	reports := []Report{}
	return reports, s.db.SelectContext(ctx, &reports, query, eventKey, realmID)
}

// GetEventTeamReportsForRealm retrieves all reports for a specific team and event, filtering to only retrieve reports for realms
// that are sharing reports or have a matching realm ID.
func (s *Service) GetEventTeamReportsForRealm(ctx context.Context, eventKey string, teamKey string, realmID *int64) (reports []Report, err error) {
	const query = `
	SELECT reports.*
	FROM reports
	INNER JOIN realms
		ON realms.id = reports.realm_id
	WHERE
		reports.event_key = $1 AND
		reports.team_key = $2 AND
		(realms.share_reports = true OR realms.id = $3)`

	reports = make([]Report, 0)
	return reports, s.db.SelectContext(ctx, &reports, query, eventKey, teamKey, realmID)
}

// GetMatchTeamReportsForRealm retrieves all reports for a specific team and event, filtering to only retrieve reports for realms
// that are sharing reports or have a matching realm ID.
func (s *Service) GetMatchTeamReportsForRealm(ctx context.Context, matchKey string, teamKey string, realmID *int64) (reports []Report, err error) {
	const query = `
	SELECT reports.*
	FROM reports
	INNER JOIN realms
		ON realms.id = reports.realm_id
	WHERE
		reports.match_key = $1 AND
		reports.team_key = $2 AND
		(realms.share_reports = true OR realms.id = $3)`

	reports = make([]Report, 0)
	return reports, s.db.SelectContext(ctx, &reports, query, matchKey, teamKey, realmID)
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
