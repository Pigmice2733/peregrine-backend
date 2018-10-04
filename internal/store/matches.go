package store

import (
	"database/sql"
)

// Match holds information about an FRC match at a specific event
type Match struct {
	Key           string
	EventKey      string
	PredictedTime *UnixTime
	ActualTime    *UnixTime
	RedScore      *int
	BlueScore     *int
	RedAlliance   []string
	BlueAlliance  []string
}

// GetTime returns the actual match time if available, and if not, predicted time
func (m *Match) GetTime() *UnixTime {
	if m.ActualTime != nil {
		return m.ActualTime
	}
	return m.PredictedTime
}

// GetEventMatches returns all matches from a specfic event.
func (s *Service) GetEventMatches(eventKey string) ([]Match, error) {
	matches := []Match{}
	rows, err := s.db.Query("SELECT key, predicted_time, actual_time, red_score, blue_score FROM matches WHERE event_key = $1", eventKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		match := Match{EventKey: eventKey, PredictedTime: &UnixTime{}, ActualTime: &UnixTime{}}
		if err := rows.Scan(&match.Key, match.PredictedTime, match.ActualTime, &match.RedScore, &match.BlueScore); err != nil {
			return nil, err
		}

		match.BlueAlliance, err = s.GetMatchAlliance(match.Key, true)
		if err != nil {
			return nil, err
		}
		match.RedAlliance, err = s.GetMatchAlliance(match.Key, false)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}

	return matches, rows.Err()
}

// GetTeamMatches returns all matches from a specfic event that include a specific team.
func (s *Service) GetTeamMatches(eventKey string, teamKey string) ([]Match, error) {
	matches := []Match{}
	rows, err := s.db.Query(`
		SELECT
		    key, predicted_time, actual_time, red_score, blue_score
		FROM matches
		INNER JOIN alliances
			ON matches.key = alliances.match_key
		WHERE matches.event_key = $1 AND $2 = ANY(alliances.team_keys)`, eventKey, teamKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		match := Match{EventKey: eventKey, PredictedTime: &UnixTime{}, ActualTime: &UnixTime{}}
		if err := rows.Scan(&match.Key, match.PredictedTime, match.ActualTime, &match.RedScore, &match.BlueScore); err != nil {
			return nil, err
		}

		match.BlueAlliance, err = s.GetMatchAlliance(match.Key, true)
		if err != nil {
			return nil, err
		}
		match.RedAlliance, err = s.GetMatchAlliance(match.Key, false)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}

	return matches, rows.Err()
}

// GetMatch returns a specfic match.
func (s *Service) GetMatch(matchKey string) (Match, error) {
	var redScore, blueScore int
	match := Match{Key: matchKey, PredictedTime: &UnixTime{}, ActualTime: &UnixTime{}}
	if err := s.db.QueryRow("SELECT event_key, predicted_time, actual_time, red_score, blue_score FROM matches WHERE key = $1", matchKey).
		Scan(&match.EventKey, match.PredictedTime, match.ActualTime, &redScore, &blueScore); err != nil {
		if err == sql.ErrNoRows {
			return match, NoResultError{err}
		}
		return match, err
	}

	match.BlueScore = &blueScore
	match.RedScore = &redScore

	var err error
	match.BlueAlliance, err = s.GetMatchAlliance(match.Key, true)
	if err != nil {
		return match, err
	}
	match.RedAlliance, err = s.GetMatchAlliance(match.Key, false)
	return match, err
}

// MatchesUpsert upserts multiple matches and their alliances into the database.
func (s *Service) MatchesUpsert(matches []Match) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO matches (key, event_key, predicted_time, actual_time, red_score, blue_score)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (key)
		DO
			UPDATE
				SET event_key = $2, predicted_time = $3, actual_time = $4, red_score = $5, blue_score = $6
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, match := range matches {
		if _, err = stmt.Exec(match.Key, match.EventKey, match.PredictedTime, match.ActualTime, match.RedScore, match.BlueScore); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = s.AlliancesUpsert(match.Key, match.BlueAlliance, match.RedAlliance, tx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
