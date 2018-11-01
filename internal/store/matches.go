package store

import (
	"database/sql"
)

// Match holds information about an FRC match at a specific event
type Match struct {
	Key           string    `json:"key" db:"key"`
	EventKey      string    `json:"eventKey" db:"event_key"`
	PredictedTime *UnixTime `json:"predictedTime" db:"predicted_time"`
	ActualTime    *UnixTime `json:"actualTime" db:"actual_time"`
	ScheduledTime *UnixTime `json:"scheduledTime" db:"scheduled_time"`
	RedScore      *int      `json:"redScore" db:"red_score"`
	BlueScore     *int      `json:"blueScore" db:"blue_score"`
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

// GetEventMatches returns all matches from a specific event.
func (s *Service) GetEventMatches(eventKey string) ([]Match, error) {
	matches := []Match{}

	err := s.db.Select(&matches, `
		SELECT
			key, predicted_time, scheduled_time, actual_time, red_score, blue_score
			FROM matches
			WHERE event_key = $1`, eventKey)
	if err != nil {
		return nil, err
	}

	for i, match := range matches {
		match.BlueAlliance, err = s.GetMatchAlliance(match.Key, true)
		if err != nil {
			return nil, err
		}

		match.RedAlliance, err = s.GetMatchAlliance(match.Key, false)
		if err != nil {
			return nil, err
		}

		matches[i] = match // value vs reference stuff
	}

	return matches, nil
}

// GetTeamMatches returns all matches from a specific event that include a specific team.
func (s *Service) GetTeamMatches(eventKey string, teamKey string) ([]Match, error) {
	matches := []Match{}
	err := s.db.Select(&matches, `
		SELECT
		    key, predicted_time, scheduled_time, actual_time, red_score, blue_score
		FROM matches
		INNER JOIN alliances
			ON matches.key = alliances.match_key
		WHERE matches.event_key = $1 AND $2 = ANY(alliances.team_keys)`, eventKey, teamKey)
	if err != nil {
		return nil, err
	}

	for i, match := range matches {
		match.BlueAlliance, err = s.GetMatchAlliance(match.Key, true)
		if err != nil {
			return nil, err
		}

		match.RedAlliance, err = s.GetMatchAlliance(match.Key, false)
		if err != nil {
			return nil, err
		}

		matches[i] = match // value vs reference stuff
	}

	return matches, nil
}

// GetMatch returns a specific match.
func (s *Service) GetMatch(matchKey string) (Match, error) {
	var m Match
	if err := s.db.Get(&m, "SELECT * FROM matches WHERE key = $1", matchKey); err != nil {
		if err == sql.ErrNoRows {
			return m, ErrNoResults(err)
		}
		return m, err
	}

	var err error
	m.BlueAlliance, err = s.GetMatchAlliance(m.Key, true)
	if err != nil {
		return m, err
	}

	m.RedAlliance, err = s.GetMatchAlliance(m.Key, false)
	return m, err
}

// MatchesUpsert upserts multiple matches and their alliances into the database.
func (s *Service) MatchesUpsert(matches []Match) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareNamed(`
		INSERT INTO matches (key, event_key, predicted_time, scheduled_time, actual_time, red_score, blue_score)
		VALUES (:key, :event_key, :predicted_time, :scheduled_time, :actual_time, :red_score, :blue_score)
		ON CONFLICT (key)
		DO
			UPDATE
				SET
					event_key = :event_key,
					predicted_time = :predicted_time,
					scheduled_time = :scheduled_time,
					actual_time = :actual_time,
					red_score = :red_score,
					blue_score = :blue_score
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, match := range matches {
		if _, err = stmt.Exec(match); err != nil {
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
