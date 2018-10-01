package store

import (
	"database/sql"

	"github.com/lib/pq"
)

// GetMatchAlliance returns a alliance from a specific match. matchKey is the
// key of the match to get the alliance from, getBlue is a boolean indicating
// whether to get the blue alliance. If getBlue is false, the red alliance will
// be retrieved instead.
func (s *Service) GetMatchAlliance(matchKey string, getBlue bool) ([]string, error) {
	var alliance []string
	err := s.db.QueryRow("SELECT team_keys FROM alliances WHERE match_key = $1 AND is_blue = $2", matchKey, getBlue).Scan(pq.Array(&alliance))
	if err == sql.ErrNoRows {
		return alliance, NoResultError{err}
	}
	return alliance, err
}

// AlliancesUpsert upserts the red and blue alliances for a specific match.
// matchKey is the key of the match. Upsert done within transaction.
func (s *Service) AlliancesUpsert(matchKey string, blueAlliance []string, redAlliance []string, tx *sql.Tx) error {
	stmt, err := tx.Prepare(`
		INSERT INTO alliances (team_keys, match_key, is_blue)
		VALUES ($1, $2, $3)
		ON CONFLICT (match_key, is_blue)
		DO
			UPDATE
				SET team_keys = $1
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(pq.Array(&blueAlliance), matchKey, true)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(pq.Array(&redAlliance), matchKey, false)
	return err
}
