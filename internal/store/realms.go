package store

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Realm holds the name of a realm, and whether to share the realms reports.
type Realm struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	ShareReports bool   `json:"shareReports" db:"share_reports"`
}

// PatchRealm is a nullable Realm, except for the ID.
type PatchRealm struct {
	ID           int64   `db:"id"`
	Name         *string `db:"name"`
	ShareReports *bool   `db:"share_reports"`
}

// GetRealms returns all realms in the database.
func (s *Service) GetRealms() ([]Realm, error) {
	realms := []Realm{}

	return realms, s.db.Select(&realms, "SELECT * FROM realms")
}

// GetPublicRealms returns all public realms in the database.
func (s *Service) GetPublicRealms() ([]Realm, error) {
	realms := []Realm{}

	return realms, s.db.Select(&realms, "SELECT * FROM realms WHERE share_reports = TRUE")
}

// GetRealm retrieves a specific realm.
func (s *Service) GetRealm(id int64) (Realm, error) {
	var realm Realm
	err := s.db.Get(&realm, "SELECT * FROM realms WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return realm, &ErrNoResults{msg: fmt.Sprintf("realm with id %d not found", id)}
	}
	return realm, err
}

// InsertRealm inserts a realm into the database.
func (s *Service) InsertRealm(realm Realm) (int64, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return -1, errors.Wrap(err, "unable to begin transaction")
	}

	stmt, err := tx.PrepareNamed(`
	    INSERT INTO realms (name, share_reports)
		    VALUES (:name, :share_reports)
	        RETURNING id
    `)
	if err != nil {
		_ = tx.Rollback()
		return -1, errors.Wrap(err, "unable to prepare realm insert statement")
	}

	err = stmt.Get(&realm.ID, realm)
	if err != nil {
		_ = tx.Rollback()
		if err, ok := err.(*pq.Error); ok && err.Code == pgExists {
			return -1, &ErrExists{msg: fmt.Sprintf("realm with name: %s already exists", realm.Name)}
		}
		return -1, errors.Wrap(err, "unable to insert realm")
	}

	return realm.ID, tx.Commit()
}

// DeleteRealm deletes a realm from the database.
func (s *Service) DeleteRealm(id int64) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.Exec(`
		DELETE FROM realms WHERE id = $1
	`, id)
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
		    share_reports = COALESCE(:share_reports, share_reports)
	    WHERE
		    id = :id
	`, realm)
	if err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to patch realm")
	}

	if count, err := result.RowsAffected(); err != nil || count == 0 {
		_ = tx.Rollback()
		return &ErrNoResults{msg: fmt.Sprintf("realm %d not found", realm.ID)}
	}

	return errors.Wrap(tx.Commit(), "unable to patch realm")
}
