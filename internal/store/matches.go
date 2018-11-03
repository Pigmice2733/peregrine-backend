package store

import (
	"database/sql"

	"github.com/lib/pq"
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

// GetMatches returns all matches from a specific event that include the given
// teams. If teams is nil or empty a list of all the matches for that event are
// returned.
func (s *Service) GetMatches(eventKey string, teamKeys []string) ([]Match, error) {
	if teamKeys == nil {
		teamKeys = []string{}
	}

	matches := []Match{}
	err := s.db.Select(&matches, `
		SELECT
		    key, predicted_time, scheduled_time, actual_time, red_score, blue_score
		FROM matches
		INNER JOIN alliances
			ON matches.key = alliances.match_key
		WHERE matches.event_key = $1 AND alliances.team_keys @> $2`, eventKey, pq.Array(teamKeys))
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

// UpsertMatch upserts a match and its alliances into the database.
func (s *Service) UpsertMatch(match Match) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`
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
	`, match)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if err = s.AlliancesUpsert(match.Key, match.BlueAlliance, match.RedAlliance, tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// UpdateTBAMatches puts a set of multiple matches and their alliances from TBA
// into the database. New matches are added, existing matches will be updated,
// and matches deleted from TBA will be deleted from the database. User-created
// matches will be unaffected. If eventKey is specified, only matches from that
// event will be affected.
func (s *Service) UpdateTBAMatches(matches []Match, eventKey string) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	upsert, err := tx.PrepareNamed(`
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
	defer upsert.Close()

	matchKeys := make([]string, len(matches))

	for i, match := range matches {
		if _, err = upsert.Exec(match); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = s.AlliancesUpsert(match.Key, match.BlueAlliance, match.RedAlliance, tx); err != nil {
			_ = tx.Rollback()
			return err
		}
		matchKeys[i] = match.Key
	}

	upsert.Close()

	if eventKey != "" {
		_, err = tx.Exec(`
		DELETE FROM matches m
			USING events e
			WHERE e.key = $1 AND
				  e.key = m.event_key AND
				  NOT e.manually_added AND
				  NOT (m.key = ANY($2)) 
	`, eventKey, pq.StringArray(matchKeys))
	} else {
		_, err = tx.Exec(`
		DELETE FROM matches m
			USING events e
			WHERE e.key = m.event_key AND
				  NOT e.manually_added AND
				  NOT (m.key = ANY($1)) 
	`, pq.StringArray(matchKeys))
	}

	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
