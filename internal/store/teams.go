package store

import (
	"database/sql"
)

// Team holds data about a single FRC team at a specific event.
type Team struct {
	Key          string
	EventKey     string
	Rank         *int
	RankingScore *float64
}

// GetTeamKeys retrieves all team keys from an event specified by eventKey.
func (s *Service) GetTeamKeys(eventKey string) ([]string, error) {
	rows, err := s.db.Query("SELECT key FROM teams WHERE event_key = $1", eventKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teamKeys := []string{}
	for rows.Next() {
		var teamKey string
		if err := rows.Scan(&teamKey); err != nil {
			return nil, err
		}
		teamKeys = append(teamKeys, teamKey)
	}

	return teamKeys, rows.Err()
}

// GetTeam retrieves a team specified by teamKey from a event specified by eventKey.
func (s *Service) GetTeam(teamKey string, eventKey string) (Team, error) {
	var rank *int
	var rankingScore *float64
	var team Team
	err := s.db.QueryRow("SELECT rank, ranking_score FROM teams WHERE key = $1 AND event_key = $2", teamKey, eventKey).Scan(&rank, &rankingScore)
	if err == sql.ErrNoRows {
		return team, NoResultError{err}
	}
	team.Key = teamKey
	team.EventKey = eventKey
	team.Rank = rank
	team.RankingScore = rankingScore
	return team, err
}

// TeamKeysUpsert upserts multiple team keys from a single event into the database.
func (s *Service) TeamKeysUpsert(eventKey string, keys []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO teams (key, event_key)
		VALUES ($1, $2)
		ON CONFLICT (key, event_key) DO NOTHING
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, key := range keys {
		if _, err = stmt.Exec(key, eventKey); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// TeamsUpsert upserts multiple teams into the database.
func (s *Service) TeamsUpsert(teams []Team) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO teams (key, event_key, rank, ranking_score)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key, event_key)
		DO
			UPDATE
				SET rank = $3, ranking_score = $4
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, team := range teams {
		if _, err = stmt.Exec(team.Key, team.EventKey, team.Rank, team.RankingScore); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
