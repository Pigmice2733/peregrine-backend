package store

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Comment defines a comment on a robots performance during a match. It is the
// qualitative equivalent of a report.
type Comment struct {
	ID         int64  `json:"id" db:"id"`
	EventKey   string `json:"-" db:"event_key"`
	MatchKey   string `json:"matchKey" db:"match_key"`
	TeamKey    string `json:"-" db:"team_key"`
	ReporterID *int64 `json:"reporterId" db:"reporter_id"`
	RealmID    *int64 `json:"-" db:"realm_id"`
	Comment    string `json:"comment" db:"comment"`
}

// UpsertMatchTeamComment will upsert a comment for a team in a match. There can only be one comment
// per reporter per team per match per event.
func (s *Service) UpsertMatchTeamComment(ctx context.Context, c Comment) (created bool, err error) {
	var existed bool

	err = s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		err = tx.QueryRow(`
			SELECT EXISTS(
				SELECT FROM comments
					WHERE
						event_key = $1 AND
						match_key = $2 AND
						team_key = $3 AND
						reporter_id = $4
			)
			`, c.EventKey, c.MatchKey, c.TeamKey, c.ReporterID).Scan(&existed)
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

// GetMatchTeamCommentsForRealm gets all comments for a given team in a match, filtering to only retrieve comments for realms
// that are sharing reports or have a matching realm ID.
func (s *Service) GetMatchTeamCommentsForRealm(ctx context.Context, matchKey, teamKey string, realmID *int64) (comments []Comment, err error) {
	const query = `
	SELECT comments.*
	FROM comments
	INNER JOIN realms
		ON realms.id = comments.realm_id
	WHERE
		comments.match_key = $1 AND
		comments.team_key = $2 AND
		(realms.share_reports = true OR realms.id = $3)`

	comments = make([]Comment, 0)
	return comments, s.db.SelectContext(ctx, &comments, query, matchKey, teamKey, realmID)
}

// GetEventTeamCommentsForRealm gets all comments for a given team in an event, filtering to only retrieve comments for realms
// that are sharing reports or have a matching realm ID.
func (s *Service) GetEventTeamCommentsForRealm(ctx context.Context, eventKey, teamKey string, realmID *int64) (comments []Comment, err error) {
	const query = `
	SELECT comments.*
	FROM comments
	INNER JOIN realms
		ON realms.id = comments.realm_id
	WHERE
		comments.event_key = $1 AND
		comments.team_key = $2 AND
		(realms.share_reports = true OR realms.id = $3)`

	comments = []Comment{}
	return comments, s.db.SelectContext(ctx, &comments, query, eventKey, teamKey, realmID)
}
