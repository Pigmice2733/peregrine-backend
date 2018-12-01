package store

import (
	"context"
	"database/sql"

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
func (s *Service) GetRealms(ctx context.Context) ([]Realm, error) {
	realms := []Realm{}

	return realms, s.db.SelectContext(ctx, &realms, "SELECT * FROM realms")
}

// GetPublicRealms returns all public realms in the database.
func (s *Service) GetPublicRealms(ctx context.Context) ([]Realm, error) {
	realms := []Realm{}

	return realms, s.db.SelectContext(ctx, &realms, "SELECT * FROM realms WHERE share_reports = TRUE")
}

// GetRealm retrieves a specific realm.
func (s *Service) GetRealm(ctx context.Context, id int64) (Realm, error) {
	var realm Realm
	err := s.db.GetContext(ctx, &realm, "SELECT * FROM realms WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return realm, ErrNoResults{errors.Wrapf(err, "realm with id %d not found", id)}
	}
	return realm, err
}

// InsertRealm inserts a realm into the database.
func (s *Service) InsertRealm(ctx context.Context, realm Realm) (int64, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, errors.Wrap(err, "unable to begin transaction")
	}

	stmt, err := tx.PrepareNamedContext(ctx, `
	    INSERT INTO realms (name, share_reports)
		    VALUES (:name, :share_reports)
	        RETURNING id
    `)
	if err != nil {
		_ = tx.Rollback()
		return 0, errors.Wrap(err, "unable to prepare realm insert statement")
	}

	err = stmt.GetContext(ctx, &realm.ID, realm)
	if err != nil {
		_ = tx.Rollback()
		if err, ok := err.(*pq.Error); ok && err.Code == pgExists {
			return 0, ErrExists{errors.Wrapf(err, "realm with name: %s already exists", realm.Name)}
		}
		return 0, errors.Wrap(err, "unable to insert realm")
	}

	return realm.ID, tx.Commit()
}

// DeleteRealm deletes a realm from the database.
func (s *Service) DeleteRealm(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM realms WHERE id = $1
	`, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// PatchRealm patches a realm.
func (s *Service) PatchRealm(ctx context.Context, realm PatchRealm) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	result, err := tx.NamedExecContext(ctx, `
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
		return ErrNoResults{errors.Wrapf(err, "realm %d not found", realm.ID)}
	}

	return errors.Wrap(tx.Commit(), "unable to patch realm")
}
