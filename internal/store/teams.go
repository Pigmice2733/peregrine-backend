package store

// Team holds data about a single FRC team at a specific event.
type Team struct {
	Key          string   `db:"key"`
	EventKey     string   `db:"event_key"`
	Rank         *int     `db:"rank"`
	RankingScore *float64 `db:"ranking_score"`
}

// GetTeamKeys retrieves all team keys from an event specified by eventKey.
func (s *Service) GetTeamKeys(eventKey string) ([]string, error) {
	teamKeys := []string{}
	return teamKeys, s.db.Select(&teamKeys, "SELECT key FROM teams WHERE event_key = $1", eventKey)
}

// GetTeam retrieves a team specified by teamKey from a event specified by eventKey.
func (s *Service) GetTeam(teamKey string, eventKey string) (Team, error) {
	var t Team
	return t, s.db.Get(&t, "SELECT * FROM teams WHERE key = $1 AND event_key = $2", teamKey, eventKey)
}

// TeamKeysUpsert upserts multiple team keys from a single event into the database.
func (s *Service) TeamKeysUpsert(eventKey string, keys []string) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO teams (key, event_key)
		VALUES ($1, $2)
		ON CONFLICT (key, event_key) DO NOTHING
	`)
	if err != nil {
		s.logErr(tx.Rollback())
		return err
	}
	defer stmt.Close()

	for _, key := range keys {
		if _, err = stmt.Exec(key, eventKey); err != nil {
			s.logErr(tx.Rollback())
			return err
		}
	}

	return tx.Commit()
}

// TeamsUpsert upserts multiple teams into the database.
func (s *Service) TeamsUpsert(teams []Team) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareNamed(`
		INSERT INTO teams (key, event_key, rank, ranking_score)
		VALUES (:key, :event_key, :rank, :ranking_score)
		ON CONFLICT (key, event_key)
		DO
			UPDATE
				SET rank = $3, ranking_score = $4
	`)
	if err != nil {
		s.logErr(tx.Rollback())
		return err
	}
	defer stmt.Close()

	for _, team := range teams {
		if _, err = stmt.Exec(team); err != nil {
			s.logErr(tx.Rollback())
			return err
		}
	}

	return tx.Commit()
}
