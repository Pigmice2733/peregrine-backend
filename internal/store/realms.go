package store

import (
	"database/sql"

	"github.com/pkg/errors"
)

// Realm holds the team key and name of a realm.
type Realm struct {
	Team       string `db:"team"`
	Name       string `db:"name"`
	PublicData bool   `db:"public_data"`
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
		return realm, ErrNoResults(err)
	}
	return realm, err
}

// UpsertRealm upserts a realm into the database.
func (s *Service) UpsertRealm(realm Realm) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.NamedExec(`
		INSERT INTO realms (team, name, public_data)
		VALUES (:team, :name, :public_data)
		ON CONFLICT (team)
		DO
			UPDATE
				SET
					name = :name,
					public_data = :public_data
	`, realm)
	if err != nil {
		_ = tx.Rollback()
		return err
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
func (s *Service) PatchRealm(realm Realm) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	if _, err := tx.NamedExec(`
	UPDATE realms
	    SET
		    name = COALESCE(:name, name),
		    public_data = COALESCE(:public_data, public_data),
	    WHERE
		    team = :team
	`, realm); err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to patch realm")
	}

	return errors.Wrap(tx.Commit(), "unable to patch realm")
}
