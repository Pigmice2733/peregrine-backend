package store

import (
	"fmt"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// SchemaOverride overrides the standard schema for TBA events on a per-realm
// basis.
type SchemaOverride struct {
	realmID  int64 `db:"realm_id"`
	schemaID int64 `db:"schema_id"`
}

// AddSchemaOverride adds a new schema override to a realm for a specific event.
func (s *Service) AddSchemaOverride(so SchemaOverride) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.NamedExec(`
	INSERT
		INTO
			schema_overrides (realm_id, schema_id)
		VALUES (:realm_id, :schema_id)
	`, so)

	if err, ok := err.(*pq.Error); ok && err.Code == pgExists {
		_ = tx.Rollback()
		return &ErrExists{msg: fmt.Sprintf("schema override already exists: %v", err.Error())}
	} else if err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to insert schema override")
	}

	return errors.Wrap(tx.Commit(), "unable to commit transaction")
}

// DeleteSchemaOverride removes a override from the database.
func (s *Service) DeleteSchemaOverride(so SchemaOverride) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.Exec(`
		DELETE FROM schema_overrides WHERE realm_id = $1 and schema_id = $2
	`, so.realmID, so.schemaID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// DeleteRealmSchemaOverrides removes all overrides from a realm.
func (s *Service) DeleteRealmSchemaOverrides(realmID int64) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.Exec(`
		DELETE FROM schema_overrides WHERE realm_id = $1
	`, realmID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
