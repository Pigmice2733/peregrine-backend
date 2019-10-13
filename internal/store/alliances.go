package store

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// AlliancesUpsertTx upserts the red and blue alliances for a specific match.
// matchKey is the key of the match. The upsert is done within the given
// transaction.
func (s *Service) AlliancesUpsertTx(ctx context.Context, tx *sqlx.Tx, eventKey, matchKey string, blueAlliance []string, redAlliance []string) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO alliances (team_keys, event_key, match_key, is_blue)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_key, match_key, is_blue)
		DO
			UPDATE
				SET team_keys = $1
	`)
	if err != nil {
		return fmt.Errorf("unable to prepare alliances upsert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, pq.Array(&blueAlliance), eventKey, matchKey, true)
	if err != nil {
		return fmt.Errorf("unable to upsert blue alliance: %w", err)
	}

	_, err = stmt.ExecContext(ctx, pq.Array(&redAlliance), eventKey, matchKey, false)
	if err != nil {
		return fmt.Errorf("unable to upsert red alliance: %w", err)
	}

	return nil
}
