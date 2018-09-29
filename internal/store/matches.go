package store

import "database/sql"

// Match holds information about an FRC match at a specific event
type Match struct {
	ID            string
	EventID       string
	PredictedTime *UnixTime
	ActualTime    *UnixTime
	RedScore      *int
	BlueScore     *int
	RedAlliance   []string
	BlueAlliance  []string
}

// GetEventMatches returns all matches from a specfic event.
func (s *Service) GetEventMatches(eventID string) ([]Match, error) {
	var matches []Match

	rows, err := s.db.Query("SELECT id, predicted_time, actual_time, red_score, blue_score FROM matches WHERE event_id = $1", eventID)
	if err != nil {
		return matches, err
	}
	defer rows.Close()

	for rows.Next() {
		match := Match{EventID: eventID, PredictedTime: &UnixTime{}, ActualTime: &UnixTime{}}
		var redScore, blueScore int
		if err := rows.Scan(&match.ID, match.PredictedTime, match.ActualTime, &redScore, &blueScore); err != nil {
			return nil, err
		}

		match.BlueScore = &blueScore
		match.RedScore = &redScore

		match.BlueAlliance, err = s.GetMatchAlliance(match.ID, true)
		if err != nil {
			return nil, err
		}
		match.RedAlliance, err = s.GetMatchAlliance(match.ID, false)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}

	return matches, rows.Err()
}

// GetMatch returns a specfic match.
func (s *Service) GetMatch(matchID string) (Match, error) {
	var redScore, blueScore int
	match := Match{ID: matchID, PredictedTime: &UnixTime{}, ActualTime: &UnixTime{}}
	if err := s.db.QueryRow("SELECT event_id, predicted_time, actual_time, red_score, blue_score FROM matches WHERE id = $1", matchID).
		Scan(&match.EventID, match.PredictedTime, match.ActualTime, &redScore, &blueScore); err != nil {
		if err == sql.ErrNoRows {
			return match, ErrNoResult
		}
		return match, err
	}

	match.BlueScore = &blueScore
	match.RedScore = &redScore

	var err error
	match.BlueAlliance, err = s.GetMatchAlliance(match.ID, true)
	if err != nil {
		return match, err
	}
	match.RedAlliance, err = s.GetMatchAlliance(match.ID, false)
	return match, err
}

// MatchesUpsert upserts multiple matches and their alliances into the database.
func (s *Service) MatchesUpsert(matches []Match) error {
	matchStmt, err := s.db.Prepare(`
		INSERT INTO matches (id, event_id, predicted_time, actual_time, red_score, blue_score)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id)
		DO
			UPDATE
				SET event_id = $2, predicted_time = $3, actual_time = $4, red_score = $5, blue_score = $6
	`)
	if err != nil {
		return err
	}
	defer matchStmt.Close()

	for _, match := range matches {
		if _, err := matchStmt.Exec(match.ID, match.EventID, match.PredictedTime, match.ActualTime, match.RedScore, match.BlueScore); err != nil {
			return err
		}
		if err := s.AlliancesUpsert(match.ID, match.BlueAlliance, match.RedAlliance); err != nil {
			return err
		}
	}

	return nil
}
