package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
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

// GetEventTeam retrieves a team specified by teamKey from an event specified by eventKey.
func (s *Service) GetEventTeam(ctx context.Context, teamKey string, eventKey string) (EventTeam, error) {
	var t EventTeam
	err := s.db.GetContext(ctx, &t, "SELECT * FROM teams WHERE key = $1 AND event_key = $2", teamKey, eventKey)
	if err == sql.ErrNoRows {
		return t, ErrNoResults{errors.Wrapf(err, "team %s at event %s does not exist", teamKey, eventKey)}
	}
	return t, err
}

// GetEventTeams retrieves all teams from an event specified by eventKey.
func (s *Service) GetEventTeams(ctx context.Context, eventKey string) ([]EventTeam, error) {
	teams := []EventTeam{}
	return teams, s.db.SelectContext(ctx, &teams, "SELECT * FROM teams WHERE event_key = $1", eventKey)
}

// GetTeam retrieves general team info for a specific team
func (s *Service) GetTeam(ctx context.Context, teamKey string) (Team, error) {
	var t Team
	err := s.db.GetContext(ctx, &t, "SELECT * FROM all_teams WHERE key = $1", teamKey)
	if err == sql.ErrNoRows {
		return t, ErrNoResults{errors.Wrapf(err, "team %s does not exist", teamKey)}
	}
	return t, err
}

// EventTeamKeysUpsert upserts multiple team keys from a single event into the database.
func (s *Service) EventTeamKeysUpsert(ctx context.Context, eventKey string, keys []string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	allTeamsStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO all_teams (key)
		VALUES ($1)
		ON CONFLICT
			DO NOTHING
	`)
	if err != nil {
		s.logErr(tx.Rollback())
		return err
	}
	defer allTeamsStmt.Close()

	for _, team := range keys {
		if _, err = allTeamsStmt.ExecContext(ctx, team); err != nil {
			s.logErr(tx.Rollback())
			return err
		}
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

// EventTeamsUpsert upserts multiple teams for a specific event into the database.
func (s *Service) EventTeamsUpsert(ctx context.Context, teams []EventTeam) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	allTeamsStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO all_teams (key)
		VALUES (:key)
		ON CONFLICT
			DO NOTHING
	`)
	if err != nil {
		s.logErr(tx.Rollback())
		return err
	}
	defer allTeamsStmt.Close()

	for _, team := range teams {
		if _, err = allTeamsStmt.ExecContext(ctx, team); err != nil {
			s.logErr(tx.Rollback())
			return err
		}
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO teams (key, event_key, rank, ranking_score)
		VALUES (:key, :event_key, :rank, :ranking_score)
		ON CONFLICT (key, event_key)
			DO UPDATE
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

// TeamsUpsert upserts multiple teams into the database.
func (s *Service) TeamsUpsert(ctx context.Context, teams []Team) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO all_teams (key, nickname)
		VALUES (:key, :nickname)
		ON CONFLICT (key)
		DO
			UPDATE
				SET nickname = $2
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
