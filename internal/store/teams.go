package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// EventTeam holds data about a single FRC team at a specific event.
type EventTeam struct {
	Key          string   `json:"team" db:"key"`
	EventKey     string   `json:"-" db:"event_key"`
	Rank         *int     `json:"rank,omitempty" db:"rank"`
	RankingScore *float64 `json:"rankingScore,omitempty" db:"ranking_score"`
}

// Team holds non-event-specific team info.
type Team struct {
	Key      string `json:"key" db:"key"`
	Nickname string `json:"nickname" db:"nickname"`
}

const allTeamsKeyUpsert = `
INSERT INTO all_teams (key)
	VALUES (:key)
	ON CONFLICT
		DO NOTHING
`

// GetEventTeamForRealm retrieves a team specified by teamKey from an event specified by eventKey with a null
// or matching realm ID.
func (s *Service) GetEventTeamForRealm(ctx context.Context, teamKey string, eventKey string, realmID *int64) (EventTeam, error) {
	var t EventTeam
	err := s.db.GetContext(ctx, &t, `
	SELECT teams.*
	FROM teams
	LEFT JOIN
		events
			ON events.key = teams.event_key
	WHERE
		teams.key = $1 AND
		event_key = $2 AND
		(events.realm_id IS NULL OR events.realm_id = $3)`, teamKey, eventKey, realmID)
	if err == sql.ErrNoRows {
		return t, ErrNoResults{fmt.Errorf("team %s at event %s does not exist: %w", teamKey, eventKey, err)}
	}
	return t, err
}

// GetEventTeamsForRealm retrieves all teams from an event specified by eventKey with a null or matching realm ID.
func (s *Service) GetEventTeamsForRealm(ctx context.Context, eventKey string, realmID *int64) ([]EventTeam, error) {
	teams := []EventTeam{}
	return teams, s.db.SelectContext(ctx, &teams, `SELECT teams.*
	FROM teams
	LEFT JOIN
		events
			ON events.key = teams.event_key
	WHERE
		event_key = $1 AND
		(events.realm_id IS NULL OR events.realm_id = $2)`, eventKey, realmID)
}

// GetTeam retrieves general team info for a specific team
func (s *Service) GetTeam(ctx context.Context, teamKey string) (Team, error) {
	var t Team
	err := s.db.GetContext(ctx, &t, "SELECT * FROM all_teams WHERE key = $1", teamKey)
	if err == sql.ErrNoRows {
		return t, ErrNoResults{fmt.Errorf("team %s does not exist: %w", teamKey, err)}
	} else if err != nil {
		return t, fmt.Errorf("unable to get team: %w", err)
	}

	return t, nil
}

// EventTeamsUpsert upserts multiple teams for a specific event into the database.
func (s *Service) EventTeamsUpsert(ctx context.Context, teams []EventTeam) error {
	return s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		allTeamsStmt, err := tx.PrepareNamedContext(ctx, allTeamsKeyUpsert)
		if err != nil {
			return fmt.Errorf("unable to prepare all_teams upsert statement: %w", err)
		}
		defer allTeamsStmt.Close()

		for _, team := range teams {
			if _, err = allTeamsStmt.ExecContext(ctx, team); err != nil {
				return fmt.Errorf("unable to upsert into all_teams: %w", err)
			}
		}

		stmt, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO teams (key, event_key, rank, ranking_score)
		VALUES (:key, :event_key, :rank, :ranking_score)
		ON CONFLICT (key, event_key)
			DO UPDATE
				SET rank = $3, ranking_score = $4
		`)
		if err != nil {
			return fmt.Errorf("unable to prepare teams upsert statement: %w", err)
		}
		defer stmt.Close()

		for _, team := range teams {
			if _, err = stmt.ExecContext(ctx, team); err != nil {
				return fmt.Errorf("unable to upsert into teams: %w", err)
			}
		}

		return nil
	})
}

// TeamsUpsert upserts multiple teams into the database.
func (s *Service) TeamsUpsert(ctx context.Context, teams []Team) error {
	return s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		stmt, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO all_teams (key, nickname)
		VALUES (:key, :nickname)
		ON CONFLICT (key)
		DO
			UPDATE
				SET nickname = $2
		`)
		if err != nil {
			return fmt.Errorf("unable to prepare all_teams upsert statement: %w", err)
		}
		defer stmt.Close()

		for _, team := range teams {
			if _, err = stmt.ExecContext(ctx, team); err != nil {
				return fmt.Errorf("unable to upsert team %q: %w", team.Key, err)
			}
		}
		return nil
	})
}

// EventTeamKeysUpsertTx upserts multiple team keys from a single event into the database in the given transaction.
func (s *Service) EventTeamKeysUpsertTx(ctx context.Context, tx *sqlx.Tx, eventKey string, keys []string) error {
	allTeamsStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO all_teams (key)
		VALUES ($1)
		ON CONFLICT
			DO NOTHING
		`)
	if err != nil {
		return fmt.Errorf("unable to prepare all teams key upsert statement: %w", err)
	}
	defer allTeamsStmt.Close()

	for _, team := range keys {
		if _, err = allTeamsStmt.ExecContext(ctx, team); err != nil {
			return fmt.Errorf("unable to upsert all team key: %w", err)
		}
	}

	stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO teams (key, event_key)
			VALUES ($1, $2)
			ON CONFLICT (key, event_key) DO NOTHING
		`)
	if err != nil {
		return fmt.Errorf("unable to prepare teams key upsert statement: %w", err)
	}
	defer stmt.Close()

	for _, key := range keys {
		if _, err = stmt.ExecContext(ctx, key, eventKey); err != nil {
			return fmt.Errorf("unable to upsert team key: %w", err)
		}
	}

	return nil
}
