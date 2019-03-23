package store

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Comment defines a comment on a robots performance during a match. It is the
// qualitative equivalent of a report.
type Comment struct {
	ID         int64  `json:"-" db:"id"`
	EventKey   string `json:"-" db:"event_key"`
	MatchKey   string `json:"-" db:"match_key"`
	TeamKey    string `json:"-" db:"team_key"`
	ReporterID *int64 `json:"reporterId" db:"reporter_id"`
	RealmID    *int64 `json:"-" db:"realm_id"`
	Comment    string `json:"comment" db:"comment"`
}

// UpsertMatchTeamComment will upsert a comment for a team in a match. There can only be one comment
// per reporter per team per match per event.
func (s *Service) UpsertMatchTeamComment(ctx context.Context, c Comment) (created bool, err error) {
	var existed bool

	err = s.doTransaction(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec("LOCK TABLE comments IN EXCLUSIVE MODE"); err != nil {
			return errors.Wrap(err, "unable to lock comments table")
		}

		err = tx.QueryRow(`
			SELECT EXISTS(
				SELECT FROM comments
					WHERE
						match_key = $1 AND
						team_key = $2
			)
			`, c.MatchKey, c.TeamKey).Scan(&existed)
		if err != nil {
			return errors.Wrap(err, "unable to check if comment exists")
		}

		_, err = tx.NamedExecContext(ctx, `
		INSERT
			INTO
				comments (event_key, match_key, team_key, reporter_id, realm_id, comment)
			VALUES (:event_key, :match_key, :team_key, :reporter_id, :realm_id, :comment)
			ON CONFLICT (event_key, match_key, team_key, reporter_id) DO
				UPDATE
					SET
						comment = :comment`, c)
		if err != nil {
			return errors.Wrap(err, "unable to upsert comment")
		}

		return nil
	})

	return !existed, err
}

// GetMatchTeamComments gets all comments for a given team in a match.
func (s *Service) GetMatchTeamComments(ctx context.Context, matchKey, teamKey string) (comments []Comment, err error) {
	comments = []Comment{}
	return comments, s.db.SelectContext(ctx, &comments, "SELECT * FROM comments WHERE match_key = $1 AND team_key = $2", matchKey, teamKey)
}

// GetEventComments gets all comments for a given team in an event.
func (s *Service) GetEventComments(ctx context.Context, eventKey, teamKey string) (comments []Comment, err error) {
	comments = []Comment{}
	return comments, s.db.SelectContext(ctx, &comments, "SELECT * FROM comments WHERE event_key = $1 AND team_key = $2", eventKey, teamKey)
}
