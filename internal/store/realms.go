package store

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Realm holds the team key and name of a realm.
type Realm struct {
	Team       string `json:"team" db:"team"`
	Name       string `json:"name" db:"name"`
	PublicData bool   `json:"publicData" db:"public_data"`
}

// PatchRealm is a nullable Realm, except for the Team, which is the PK.
type PatchRealm struct {
	Team       string  `db:"team"`
	Name       *string `db:"name"`
	PublicData *bool   `db:"public_data"`
}

// GetRealms returns all realms in the database.
func (s *Service) GetRealms() ([]Realm, error) {
	realms := []Realm{}

	return realms, s.db.Select(&realms, "SELECT * FROM realms")
}

// GetRealm retrieves a specific realm.
func (s *Service) GetRealm(team string) (Realm, error) {
	var realm Realm
	err := s.db.Get(&realm, "SELECT * FROM realms WHERE team = $1", team)
	if err == sql.ErrNoRows {
		return realm, ErrNoResults
	}
	return realm, err
}

// InsertRealm inserts a realm into the database.
func (s *Service) InsertRealm(realm Realm) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.NamedExec(`
		INSERT INTO realms (team, name, public_data)
		VALUES (:team, :name, :public_data)
	`, realm)
	if err != nil {
		if err, ok := err.(*pq.Error); ok && err.Code == pgExists {
			_ = tx.Rollback()
			return ErrExists
		}
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to insert realm")
	}

	return tx.Commit()
}

// DeleteRealm deletes a realm from the database.
func (s *Service) DeleteRealm(team string) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.Exec(`
		DELETE FROM realms WHERE team = $1
	`, team)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// PatchRealm patches a realm.
func (s *Service) PatchRealm(realm PatchRealm) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	result, err := tx.NamedExec(`
	UPDATE realms
	    SET
		    name = COALESCE(:name, name),
		    public_data = COALESCE(:public_data, public_data)
	    WHERE
		    team = :team
	`, realm)
	if err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to patch realm")
	}

	if count, err := result.RowsAffected(); err != nil || count == 0 {
		_ = tx.Rollback()
		return ErrNoResults
	}

	return errors.Wrap(tx.Commit(), "unable to patch realm")
}
