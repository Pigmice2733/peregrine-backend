package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
	ID         int64      `json:"id" db:"id"`
	EventKey   string     `json:"eventKey" db:"event_key"`
	MatchKey   string     `json:"matchKey" db:"match_key"`
	TeamKey    string     `json:"teamKey" db:"team_key"`
	ReporterID *int64     `json:"reporterId" db:"reporter_id"`
	RealmID    *int64     `json:"realmId" db:"realm_id"`
	Data       ReportData `json:"data" db:"data"`
	Comment    string     `json:"comment" db:"comment"`
}

// Leaderboard holds information about how many reports each reporter submitted.
type Leaderboard []struct {
	ReporterID int64 `json:"reporterId" db:"reporter_id"`
	Reports    int64 `json:"reports" db:"num_reports"`
}

// LockReport retrieves a report and locks it for update
func (s *Service) LockReport(ctx context.Context, tx *sqlx.Tx, id int64) (Report, error) {
	var report Report

	err := tx.GetContext(ctx, &report, "SELECT * FROM reports WHERE id = $1 FOR UPDATE", id)
	if err == sql.ErrNoRows {
		return report, ErrNoResults{fmt.Errorf("report with ID %d does not exist", report.ID)}
	} else if err != nil {
		return report, fmt.Errorf("unable to retrieve report: %w", err)
	}

	return report, nil
}

// GetReportForRealm retrieves a report in a specific realm
func (s *Service) GetReportForRealm(ctx context.Context, id int64, realmID *int64) (Report, error) {
	var report Report

	err := s.db.GetContext(ctx, &report, `
	SELECT reports.*
		FROM reports
	LEFT JOIN realms
		ON realms.id = reports.realm_id
	WHERE
		reports.id = $1 AND
		(reports.realm_id IS NULL OR realms.share_reports = true OR realms.id = $2)
	`, id, realmID)
	if err == sql.ErrNoRows {
		return report, ErrNoResults{fmt.Errorf("report with ID %d does not exist", report.ID)}
	} else if err != nil {
		return report, fmt.Errorf("unable to retrieve report: %w", err)
	}

	return report, nil
}

// UpsertReport creates a new report in the db, or replaces the existing one if
// the same reporter already has a report in the db for that team and match. It
// returns a boolean that is true when the report was created, and false when it
// was updated.
func (s *Service) UpsertReport(ctx context.Context, r Report) (created bool, id int64, err error) {
	var existed bool

	err = s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
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

		reportStmt, err := tx.PrepareNamedContext(ctx, `INSERT INTO
				reports (event_key, match_key, team_key, reporter_id, realm_id, data, comment)
			VALUES (:event_key, :match_key, :team_key, :reporter_id, :realm_id, :data, :comment)
			ON CONFLICT (event_key, match_key, team_key, reporter_id)
				DO UPDATE SET data = :data, realm_id = :realm_id, comment = :comment
			RETURNING id
		`)
		if err != nil {
			return fmt.Errorf("unable to prepare user insert statement: %w", err)
		}

		err = reportStmt.GetContext(ctx, &id, r)
		if err != nil {
			if err, ok := err.(*pq.Error); ok {
				if err.Code == pgExists {
					return ErrExists{fmt.Errorf("report unique violation: %s, %s, %s, %d", r.EventKey, r.MatchKey, r.TeamKey, r.ReporterID)}
				}
				if err.Code == pgFKeyViolation {
					return ErrFKeyViolation{fmt.Errorf("report fk violation %s", err.Constraint)}
				}
			}
			return fmt.Errorf("unable to upsert report: %w", err)
		}

		return nil
	})

	return !existed, id, err
}

// ErrConflictingReport is returned when an existing report conflicts with the report we're trying
// to update.
type ErrConflictingReport struct {
	ID int64
}

// Is returns whether the target is an ErrConflictingReport.
func (err ErrConflictingReport) Is(target error) bool {
	_, ok := target.(ErrConflictingReport)
	return ok
}

func (err ErrConflictingReport) Error() string {
	return fmt.Sprintf("report with same event, match, team, and reporter id exists (id %d)", err.ID)
}

// UpdateReportTx updates an existing report in the db
func (s *Service) UpdateReportTx(ctx context.Context, tx *sqlx.Tx, r Report, replace bool) error {
	var id int64
	err := tx.GetContext(ctx, &id, `
		SELECT id
		FROM reports
		WHERE
			id != $1 AND
			event_key = $2 AND
			match_key = $3 AND
			team_key = $4 AND
			reporter_id = $5`, r.ID, r.EventKey, r.MatchKey, r.TeamKey, r.ReporterID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("unable to check if report exists: %w", err)
	} else if err == nil && !replace {
		return ErrConflictingReport{ID: id}
	} else if err == nil && replace {
		_, err := tx.ExecContext(ctx, "DELETE FROM reports WHERE id = $1", id)
		if err != nil {
			return fmt.Errorf("unable to delete conflicting report: %w", err)
		}
	}

	res, err := tx.NamedExecContext(ctx, `UPDATE reports
	SET
		event_key = :event_key,
		match_key = :match_key,
		team_key = :team_key,
		reporter_id = :reporter_id,
		realm_id = :realm_id,
		data = :data,
		comment = :comment
	WHERE
		id = :id`, r)
	if err == nil {
		if n, err := res.RowsAffected(); err != nil && n == 0 {
			return ErrNoResults{fmt.Errorf("could not update non-existent report: %w", err)}
		}
	} else if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == pgExists {
				return ErrExists{fmt.Errorf("report unique violation: %s, %s, %s, %d", r.EventKey, r.MatchKey, r.TeamKey, r.ReporterID)}
			}
			if err.Code == pgFKeyViolation {
				return ErrFKeyViolation{fmt.Errorf("report fk violation %s", err.Constraint)}
			}
		}
		return fmt.Errorf("unable to update report: %w", err)
	}

	return nil
}

// GetReports returns all reports matching the specified filters
func (s *Service) GetReports(ctx context.Context, eventKey *string, matchKey *string, teamKey *string, realmID *int64, reporterID *int64) ([]Report, error) {
	var query = `
	SELECT reports.*
	FROM reports
	LEFT JOIN realms
		ON realms.id = reports.realm_id
	WHERE
	`

	var parameters []interface{}
	filters := 1

	if realmID != nil {
		query += `(reports.realm_id IS NULL OR realms.share_reports = true OR realms.id = $1)`
		filters++
		parameters = append(parameters, *realmID)
	} else {
		query += `(reports.realm_id IS NULL OR realms.share_reports = true)`
	}

	if eventKey != nil {
		query += fmt.Sprintf(` AND reports.event_key = $%d`, filters)
		filters++
		parameters = append(parameters, *eventKey)
	}

	if matchKey != nil {
		query += fmt.Sprintf(` AND reports.match_key = $%d`, filters)
		filters++
		parameters = append(parameters, *matchKey)
	}

	if teamKey != nil {
		query += fmt.Sprintf(` AND reports.team_key = $%d`, filters)
		filters++
		parameters = append(parameters, *teamKey)
	}

	if reporterID != nil {
		query += fmt.Sprintf(` AND reports.reporter_id = $%d`, filters)
		filters++
		parameters = append(parameters, *reporterID)
	}

	reports := []Report{}
	return reports, s.db.SelectContext(ctx, &reports, query, parameters...)
}

// GetEventReportsForRealm returns all event reports for a specific event and realm.
func (s *Service) GetEventReportsForRealm(ctx context.Context, eventKey string, realmID *int64) ([]Report, error) {
	const query = `
	SELECT reports.*
	FROM reports
	INNER JOIN realms
		ON realms.id = reports.realm_id AND
		(realms.share_reports = true OR realms.id = $2)
	WHERE
		reports.event_key = $1`

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
		ON realms.id = reports.realm_id AND
		(realms.share_reports = true OR realms.id = $3)
	WHERE
		reports.event_key = $1 AND
		reports.team_key = $2`

	reports = make([]Report, 0)
	return reports, s.db.SelectContext(ctx, &reports, query, eventKey, teamKey, realmID)
}

// GetMatchTeamReportsForRealm retrieves all reports for a specific match, team, and event, filtering to only retrieve reports for realms
// that are sharing reports or have a matching realm ID.
func (s *Service) GetMatchTeamReportsForRealm(ctx context.Context, eventKey, matchKey string, teamKey string, realmID *int64) (reports []Report, err error) {
	const query = `
	SELECT reports.*
	FROM reports
	INNER JOIN realms
		ON realms.id = reports.realm_id AND
		(realms.share_reports = true OR realms.id = $4)
	WHERE
		reports.event_key = $1 AND
		reports.match_key = $2 AND
		reports.team_key = $3`

	reports = make([]Report, 0)
	return reports, s.db.SelectContext(ctx, &reports, query, eventKey, matchKey, teamKey, realmID)
}

// DeleteReportTx deletes specified report from the database using the given transaction.
func (s *Service) DeleteReportTx(ctx context.Context, tx *sqlx.Tx, id int64) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM reports WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("unable to delete report: %w", err)
	}

	return nil
}

// GetLeaderboardForRealm retrieves leaderboard information from the reports and users table for users
// in the given realm. Specify year to filter for reports for events in the given year. Leave unspecified
// for all years.
func (s *Service) GetLeaderboardForRealm(ctx context.Context, realmID int64, year *int) (Leaderboard, error) {
	leaderboard := make(Leaderboard, 0)

	return leaderboard, s.db.SelectContext(ctx, &leaderboard, `
	SELECT
		users.id AS reporter_id, COUNT(reports.reporter_id) AS num_reports
	FROM users
	LEFT JOIN reports
		ON (users.id = reports.reporter_id)
	LEFT JOIN events
		ON (reports.event_key = events.key)
	WHERE
		users.realm_id = $1 AND
		(EXTRACT(YEAR FROM events.start_date) = $2 OR $2 IS NULL)
	GROUP BY users.id
	ORDER BY num_reports DESC;
	`, realmID, year)
}
