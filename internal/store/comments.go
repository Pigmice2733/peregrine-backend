package store

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Comment defines a comment on a robots performance during a match. It is the
// qualitative equivalent of a report.
type Comment struct {
	ID       int64  `json:"id" db:"id"`
	ReportID int64  `json:"report_id" db:"report_id"`
	Comment  string `json:"comment" db:"comment"`
}

// UpsertMatchTeamComment will upsert a comment for a team in a match. There can only be one comment
// per reporter per team per match per event.
func (s *Service) UpsertMatchTeamComment(ctx context.Context, eventKey, matchKey, teamKey string, reporterID int64, comment string) (created bool, err error) {
	var existed bool

	err = s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		err = tx.QueryRow(`
			SELECT EXISTS(
				SELECT FROM comments
				INNER JOIN reports
					ON reports.id = comments.report_id
				WHERE
					reports.event_key = $1 AND
					reports.match_key = $2 AND
					reports.team_key = $3 AND
					reports.reporter_id = $4
			)
			`, eventKey, matchKey, teamKey, reporterID).Scan(&existed)
		if err != nil {
			return fmt.Errorf("unable to check if comment exists: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
		INSERT INTO comments (report_id, comment)
		SELECT reports.id, $1
			FROM reports
			WHERE
				reports.event_key = $2 AND
				reports.match_key = $3 AND
				reports.team_key = $4 AND
				reports.reporter_id = $5
		ON CONFLICT (report_id)
			DO UPDATE SET comment = $1
		`, comment, eventKey, matchKey, teamKey, reporterID)
		if err != nil {
			return fmt.Errorf("unable to upsert comment: %w", err)
		}

		return nil
	})

	return !existed, err
}

// GetMatchTeamCommentsForRealm gets all comments for a given team in a match, filtering to only retrieve comments for realms
// that are sharing reports or have a matching realm ID.
func (s *Service) GetMatchTeamCommentsForRealm(ctx context.Context, eventKey, matchKey, teamKey string, realmID *int64) (comments []Comment, err error) {
	const query = `
	SELECT comments.* FROM comments
	INNER JOIN reports
	    ON reports.id = comments.report_id
	LEFT JOIN realms
		ON realms.id = reports.realm_id AND
		(realms.share_reports = true OR realms.id = $4)
	WHERE
	    reports.event_key = $1 AND
		reports.match_key = $2 AND
		reports.team_key = $3`

	comments = make([]Comment, 0)
	return comments, s.db.SelectContext(ctx, &comments, query, eventKey, matchKey, teamKey, realmID)
}

// GetEventTeamCommentsForRealm gets all comments for a given team in an event, filtering to only retrieve comments for realms
// that are sharing reports or have a matching realm ID.
func (s *Service) GetEventTeamCommentsForRealm(ctx context.Context, eventKey, teamKey string, realmID *int64) (comments []Comment, err error) {
	const query = `
	SELECT comments.* FROM comments
	INNER JOIN reports
	    ON reports.id = comments.report_id
	LEFT JOIN realms
		ON realms.id = reports.realm_id AND
		(realms.share_reports = true OR realms.id = $3)
	WHERE
		reports.event_key = $1 AND
		reports.team_key = $2`

	comments = []Comment{}
	return comments, s.db.SelectContext(ctx, &comments, query, eventKey, teamKey, realmID)
}
