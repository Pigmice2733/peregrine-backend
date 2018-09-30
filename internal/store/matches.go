package store

import "database/sql"

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

// GetEventMatches returns all matches from a specfic event.
func (s *Service) GetEventMatches(eventKey string) ([]Match, error) {
	var matches []Match

	rows, err := s.db.Query("SELECT key, predicted_time, actual_time, red_score, blue_score FROM matches WHERE event_key = $1", eventKey)
	if err != nil {
		return matches, err
	}
	defer rows.Close()

	for rows.Next() {
		match := Match{EventKey: eventKey, PredictedTime: &UnixTime{}, ActualTime: &UnixTime{}}
		var redScore, blueScore int
		if err := rows.Scan(&match.Key, match.PredictedTime, match.ActualTime, &redScore, &blueScore); err != nil {
			return nil, err
		}

		match.BlueScore = &blueScore
		match.RedScore = &redScore

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
			return match, ErrNoResult
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
func (s *Service) MatchesUpsert(matches []Match) (err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
		err = tx.Commit()
	}()

	matchStmt, err := tx.Prepare(`
		INSERT INTO matches (key, event_key, predicted_time, actual_time, red_score, blue_score)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (key)
		DO
			UPDATE
				SET event_key = $2, predicted_time = $3, actual_time = $4, red_score = $5, blue_score = $6
	`)
	if err != nil {
		return
	}
	defer matchStmt.Close()

	for _, match := range matches {
		if _, err = matchStmt.Exec(match.Key, match.EventKey, match.PredictedTime, match.ActualTime, match.RedScore, match.BlueScore); err != nil {
			return
		}
		if err = s.AlliancesUpsert(match.Key, match.BlueAlliance, match.RedAlliance, tx); err != nil {
			return
		}
	}

	return
}
