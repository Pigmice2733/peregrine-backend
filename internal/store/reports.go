package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"

	"github.com/jmoiron/sqlx"
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
		err = tx.QueryRow(`
			SELECT EXISTS(
				SELECT FROM reports
				WHERE
					match_key = $1 AND
					team_key = $2 AND
					reporter_id = $3
			)
			`, r.MatchKey, r.TeamKey, r.ReporterID).Scan(&existed)
		if err != nil {
			return errors.Wrap(err, "unable to determine if report exists")
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

		return errors.Wrap(err, "unable to upsert report")
	})

	return !existed, err
}

// GetEventReports returns all event reports for a specific event. This returns all reports without filtering for a realm,
// so it should only be used for super-admins.
func (s *Service) GetEventReports(ctx context.Context, eventKey string) ([]Report, error) {
	const query = `
	SELECT *
	FROM reports
	WHERE
		event_key = $1`

	reports := []Report{}
	return reports, s.db.SelectContext(ctx, &reports, query, eventKey)
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

// GetMatchTeamReports retrieves all reports for a specific team and match from the db. This returns all reports without
// filtering for a realm, so it should only be used for super-admins.
func (s *Service) GetMatchTeamReports(ctx context.Context, matchKey string, teamKey string) ([]Report, error) {
	const query = `
	SELECT *
	FROM reports
	WHERE
		match_key = $1 AND
		team_key = $2`
	reports := []Report{}
	return reports, s.db.SelectContext(ctx, &reports, query, matchKey, teamKey)
}

// GetEventTeamReports retrieves all reports for a specific team and match from the db. This returns all reports without
// filtering for a realm, so it should only be used for super-admins.
func (s *Service) GetEventTeamReports(ctx context.Context, eventKey string, teamKey string) ([]Report, error) {
	const query = `
	SELECT *
	FROM reports
	WHERE
		event_key = $1 AND
		team_key = $2`
	reports := []Report{}
	return reports, s.db.SelectContext(ctx, &reports, query, eventKey, teamKey)
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
