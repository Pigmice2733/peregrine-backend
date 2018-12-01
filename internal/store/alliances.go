package store

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// AlliancesUpsert upserts the red and blue alliances for a specific match.
// matchKey is the key of the match. Upsert done within transaction.
func (s *Service) AlliancesUpsert(ctx context.Context, matchKey string, blueAlliance []string, redAlliance []string, tx *sqlx.Tx) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO alliances (team_keys, match_key, is_blue)
		VALUES ($1, $2, $3)
		ON CONFLICT (match_key, is_blue)
		DO
			UPDATE
				SET team_keys = $1
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, pq.Array(&blueAlliance), matchKey, true)
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(ctx, pq.Array(&redAlliance), matchKey, false)
	return err
}
