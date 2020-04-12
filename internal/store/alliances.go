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

// IsTeamPresentAtMatch returns whether the specified team is on an alliance in the specified match
func (s *Service) IsTeamPresentAtMatch(ctx context.Context, eventKey, matchKey, teamKey string, realmID *int64) (present bool, err error) {
	err = s.db.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT FROM alliances
				LEFT JOIN events
					ON events.key = alliances.event_key
				WHERE
					alliances.event_key = $1 AND
					alliances.match_key = $2 AND
					$3 = ANY(alliances.team_keys) AND
					(events.realm_id IS NULL OR events.realm_id = $4)

			)
			`, eventKey, matchKey, teamKey, realmID).Scan(&present)
	if err != nil {
		return present, fmt.Errorf("unable to determine if team is present: %w", err)
	}

	return present, nil
}
