package store

import (
	"github.com/lib/pq"
)

// Match holds information about an FRC match at a specific event
type Match struct {
	Key           string         `json:"key" db:"key"`
	EventKey      string         `json:"eventKey" db:"event_key"`
	PredictedTime *UnixTime      `json:"predictedTime" db:"predicted_time"`
	ActualTime    *UnixTime      `json:"actualTime" db:"actual_time"`
	ScheduledTime *UnixTime      `json:"scheduledTime" db:"scheduled_time"`
	RedScore      *int           `json:"redScore" db:"red_score"`
	BlueScore     *int           `json:"blueScore" db:"blue_score"`
	RedAlliance   pq.StringArray `json:"redAlliance" db:"red_alliance"`
	BlueAlliance  pq.StringArray `json:"blueAlliance" db:"blue_alliance"`
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
		key,
		predicted_time,
		scheduled_time,
		actual_time,
		blue_score,
		red_score,
		r.team_keys AS red_alliance,
		b.team_keys AS blue_alliance
	FROM
		matches
	INNER JOIN
		alliances r
	ON
		matches.key = r.match_key AND r.is_blue = false
	INNER JOIN
		alliances b
	ON
		matches.key = b.match_key AND b.is_blue = true
	WHERE
		matches.event_key = $1 AND
		(r.team_keys || b.team_keys) @> $2
			`, eventKey, pq.Array(teamKeys))
	if err != nil {
		return nil, err
	}

	return matches, nil
}

// GetMatch returns a specific match.
func (s *Service) GetMatch(matchKey string) (Match, error) {
	var m Match
	err := s.db.Get(&m, `
	SELECT
		key,
		predicted_time,
		scheduled_time,
		actual_time,
		blue_score,
		red_score,
		r.team_keys AS red_alliance,
		b.team_keys AS blue_alliance
	FROM
		matches
	INNER JOIN
		alliances r
	ON
		matches.key = r.match_key AND r.is_blue = false
	INNER JOIN
		alliances b
	ON
		matches.key = b.match_key AND b.is_blue = true
	WHERE
		matches.key = $1`, matchKey)

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
