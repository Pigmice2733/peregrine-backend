package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Realm holds the name of a realm, and whether to share the realms reports.
type Realm struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name" validate:"omitempty,gte=1,lte=32"`
	ShareReports bool   `json:"shareReports" db:"share_reports"`
}

// GetRealms returns all realms in the database.
func (s *Service) GetRealms(ctx context.Context) (realms []Realm, err error) {
	realms = make([]Realm, 0)
	return realms, s.db.SelectContext(ctx, &realms, "SELECT * FROM realms")
}

// GetRealm retrieves a specific realm.
func (s *Service) GetRealm(ctx context.Context, id int64) (realm Realm, err error) {
	err = s.db.GetContext(ctx, &realm, "SELECT * FROM realms WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return realm, ErrNoResults{fmt.Errorf("realm with id %d not found: %w", id, err)}
	}

	return realm, err
}

// GetRealmExistsTx returns whether the given realm exists using the given transaction.
func (s *Service) GetRealmExistsTx(ctx context.Context, tx *sqlx.Tx, id int64) (exists bool, err error) {
	err = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT id FROM realms WHERE id = $1)", id)
	if err != nil {
		return false, fmt.Errorf("unable to get whether realm %d exists: %w", id, err)
	}

	return exists, nil
}

// ExclusiveLockRealmsTx locks the entire realm table while doing an update.
func (s *Service) ExclusiveLockRealmsTx(ctx context.Context, tx *sqlx.Tx) error {
	_, err := tx.ExecContext(ctx, "LOCK TABLE realms IN EXCLUSIVE MODE")
	if err != nil {
		return fmt.Errorf("unable to lock realms: %w", err)
	}

	return nil
}

// InsertRealm inserts a realm into the database.
func (s *Service) InsertRealm(ctx context.Context, realm Realm) (int64, error) {
	var realmID int64

	err := s.db.GetContext(ctx, &realmID, `
	    INSERT INTO realms (name, share_reports)
		    VALUES ($1, $2)
	        RETURNING id
		`, realm.Name, realm.ShareReports)
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == pgExists {
		return 0, ErrExists{fmt.Errorf("realm with name: %s already exists: %w", realm.Name, err)}
	} else if err != nil {
		return 0, fmt.Errorf("unable to insert realm: %w", err)
	}

	return realmID, nil
}

// DeleteRealmTx deletes a realm from the database using the given transaction.
func (s *Service) DeleteRealmTx(ctx context.Context, tx *sqlx.Tx, id int64) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM realms WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("unable to delete realm: %w", err)
	}

	return nil
}

// UpdateRealmTx updates a realm using the given transaction.
func (s *Service) UpdateRealmTx(ctx context.Context, tx *sqlx.Tx, realm Realm) error {
	res, err := tx.NamedExecContext(ctx, `
	UPDATE realms
	    SET
		    name = :name,
		    share_reports = :share_reports
	    WHERE
		    id = :id
	`, realm)
	if err == nil {
		if n, err := res.RowsAffected(); err != nil && n == 0 {
			return ErrNoResults{fmt.Errorf("could not update non-existent realm: %w", err)}
		}
	} else if err != nil {
		return fmt.Errorf("unable to update realm: %w", err)
	}

	return nil
}
