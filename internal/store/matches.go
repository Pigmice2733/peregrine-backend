package store

import (
	"database/sql"
)

// Match holds information about an FRC match at a specific event
type Match struct {
	Key           string    `json:"key"`
	EventKey      string    `json:"eventKey"`
	PredictedTime *UnixTime `json:"predictedTime"`
	ActualTime    *UnixTime `json:"actualTime"`
	ScheduledTime *UnixTime `json:"scheduledTime"`
	RedScore      *int      `json:"redScore"`
	BlueScore     *int      `json:"blueScore"`
	RedAlliance   []string  `json:"redAlliance"`
	BlueAlliance  []string  `json:"blueAlliance"`
}

// GetTime returns the actual match time if available, and if not, predicted time
func (m *Match) GetTime() *UnixTime {
	if m.ActualTime != nil {
		return m.ActualTime
	}
	if m.PredictedTime != nil {
		return m.PredictedTime
	}
	return m.ScheduledTime
}

// GetEventMatches returns all matches from a specfic event.
func (s *Service) GetEventMatches(eventKey string) ([]Match, error) {
	var matches []Match
	rows, err := s.db.Query(`
	    SELECT
            key, predicted_time, scheduled_time, actual_time, red_score, blue_score
		FROM matches
		WHERE event_key = $1`, eventKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		match := Match{EventKey: eventKey, PredictedTime: &UnixTime{}, ActualTime: &UnixTime{}, ScheduledTime: &UnixTime{}}
		if err := rows.Scan(&match.Key, &match.PredictedTime, &match.ScheduledTime, &match.ActualTime, &match.RedScore, &match.BlueScore); err != nil {
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
	var matches []Match
	rows, err := s.db.Query(`
		SELECT
		    key, predicted_time, scheduled_time, actual_time, red_score, blue_score
		FROM matches
		INNER JOIN alliances
			ON matches.key = alliances.match_key
		WHERE matches.event_key = $1 AND $2 = ANY(alliances.team_keys)`, eventKey, teamKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		match := Match{EventKey: eventKey, PredictedTime: &UnixTime{}, ActualTime: &UnixTime{}, ScheduledTime: &UnixTime{}}
		if err := rows.Scan(&match.Key, &match.PredictedTime, &match.ScheduledTime, &match.ActualTime, &match.RedScore, &match.BlueScore); err != nil {
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
	var redScore, blueScore *int
	match := Match{Key: matchKey, PredictedTime: &UnixTime{}, ActualTime: &UnixTime{}, ScheduledTime: &UnixTime{}}
	if err := s.db.QueryRow("SELECT event_key, predicted_time, scheduled_time, actual_time, red_score, blue_score FROM matches WHERE key = $1", matchKey).
		Scan(&match.EventKey, &match.PredictedTime, &match.ScheduledTime, &match.ActualTime, &redScore, &blueScore); err != nil {
		if err == sql.ErrNoRows {
			return match, ErrNoResults(err)
		}
		return match, err
	}

	match.BlueScore = blueScore
	match.RedScore = redScore

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
		INSERT INTO matches (key, event_key, predicted_time, scheduled_time, actual_time, red_score, blue_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (key)
		DO
			UPDATE
				SET event_key = $2, predicted_time = $3, scheduled_time = $4, actual_time = $5, red_score = $6, blue_score = $7
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, match := range matches {
		if _, err = stmt.Exec(match.Key, match.EventKey, match.PredictedTime, match.ScheduledTime, match.ActualTime, match.RedScore, match.BlueScore); err != nil {
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
