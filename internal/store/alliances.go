package store

// Alliance holds information about an alliance at a specific match.
type Alliance []string

// GetMatchAlliance returns a alliance from a specific match. matchID is the
// ID of the match to get the alliance from, getBlue is a boolean indicating whether to
// get the blue alliance. If getBlue is false, the red alliance will be
// retrieved instead.
func (s *Service) GetMatchAlliance(matchID string, getBlue bool) (Alliance, error) {
	var alliance Alliance
	err := s.db.QueryRow("SELECT team_keys FROM alliances WHERE match_id = $1 AND is_blue = $2", matchID, getBlue).Scan(alliance)
	return alliance, err
}

// AlliancesUpsert upserts the red and blue alliances for a specific match. matchID is the ID of the match.
func (s *Service) AlliancesUpsert(matchID string, blue Alliance, red Alliance) error {
	allianceStmt, err := s.db.Prepare(`
		INSERT INTO alliances (team_keys, match_id, is_blue)
		VALUES ($1, $2, $3)
		ON CONFLICT (match_id, is_blue)
		DO
			UPDATE
				SET team_keys = $1
	`)
	if err != nil {
		return err
	}
	defer allianceStmt.Close()

	_, err = allianceStmt.Exec(blue, matchID, true)
	if err != nil {
		return err
	}
	_, err = allianceStmt.Exec(red, matchID, false)
	return err
}
