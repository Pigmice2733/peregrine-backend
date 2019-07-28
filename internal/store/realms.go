package store

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Realm holds the name of a realm, and whether to share the realms reports.
type Realm struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name" validate:"omitempty,gte=1,lte=32"`
	ShareReports bool   `json:"shareReports" db:"share_reports"`
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
	var realmID int64

	err := s.doTransaction(ctx, func(tx *sqlx.Tx) error {
		err := tx.GetContext(ctx, &realm.ID, `
	    INSERT INTO realms (name, share_reports)
		    VALUES (:name, :share_reports)
	        RETURNING id
		`, realm)
		if err, ok := err.(*pq.Error); ok && err.Code == pgExists {
			return ErrExists{errors.Wrapf(err, "realm with name: %s already exists", realm.Name)}
		}
		return errors.Wrap(err, "unable to insert realm")
	})

	return realmID, err
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
		s.logErr(tx.Rollback())
		return err
	}

	return tx.Commit()
}

// UpdateRealm updates a realm.
func (s *Service) UpdateRealm(ctx context.Context, realm Realm) error {
	res, err := s.db.NamedExecContext(ctx, `
	UPDATE realms
	    SET
		    name = :name,
		    share_reports = :share_reports
	    WHERE
		    id = :id
	`, realm)
	if err == nil {
		if n, err := res.RowsAffected(); err != nil && n == 0 {
			return ErrNoResults{errors.Wrap(err, "could not update non-existent realm")}
		}
	}

	return errors.Wrap(err, "unable to update realm")
}
