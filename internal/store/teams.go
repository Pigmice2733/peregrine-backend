package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

// Team holds data about a single FRC team at a specific event.
type Team struct {
	Key          string   `json:"team" db:"key"`
	EventKey     string   `json:"-" db:"event_key"`
	Rank         *int     `json:"rank,omitempty" db:"rank"`
	RankingScore *float64 `json:"rankingScore,omitempty" db:"ranking_score"`
}

// GetTeamKeys retrieves all team keys from an event specified by eventKey.
func (s *Service) GetTeamKeys(ctx context.Context, eventKey string) ([]string, error) {
	teamKeys := []string{}
	return teamKeys, s.db.SelectContext(ctx, &teamKeys, "SELECT key FROM teams WHERE event_key = $1", eventKey)
}

// GetTeam retrieves a team specified by teamKey from an event specified by eventKey.
func (s *Service) GetTeam(ctx context.Context, teamKey string, eventKey string) (Team, error) {
	var t Team
	err := s.db.GetContext(ctx, &t, "SELECT * FROM teams WHERE key = $1 AND event_key = $2", teamKey, eventKey)
	if err == sql.ErrNoRows {
		return t, ErrNoResults{errors.Wrapf(err, "team %s at event %s does not exist", teamKey, eventKey)}
	}
	return t, err
}

// GetTeams retrieves all teams from an event specified by eventKey.
func (s *Service) GetTeams(ctx context.Context, eventKey string) ([]Team, error) {
	teams := []Team{}
	return teams, s.db.SelectContext(ctx, &teams, "SELECT * FROM teams WHERE event_key = $1", eventKey)
}

// TeamKeysUpsert upserts multiple team keys from a single event into the database.
func (s *Service) TeamKeysUpsert(ctx context.Context, eventKey string, keys []string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO teams (key, event_key)
		VALUES ($1, $2)
		ON CONFLICT (key, event_key) DO NOTHING
	`)
	if err != nil {
		s.logErr(tx.Rollback())
		return err
	}
	defer stmt.Close()

	for _, key := range keys {
		if _, err = stmt.ExecContext(ctx, key, eventKey); err != nil {
			s.logErr(tx.Rollback())
			return err
		}
	}

	return tx.Commit()
}

// TeamsUpsert upserts multiple teams into the database.
func (s *Service) TeamsUpsert(ctx context.Context, teams []Team) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO teams (key, event_key, rank, ranking_score)
		VALUES (:key, :event_key, :rank, :ranking_score)
		ON CONFLICT (key, event_key)
		DO
			UPDATE
				SET rank = $3, ranking_score = $4
	`)
	if err != nil {
		s.logErr(tx.Rollback())
		return err
	}
	defer stmt.Close()

	for _, team := range teams {
		if _, err = stmt.ExecContext(ctx, team); err != nil {
			s.logErr(tx.Rollback())
			return err
		}
	}

	return tx.Commit()
}
