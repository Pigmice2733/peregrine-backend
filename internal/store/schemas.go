package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Schema describes the statistics that reports should include
type Schema struct {
	ID      int64           `json:"id" db:"id"`
	Year    *int64          `json:"year,omitempty" db:"year"`
	RealmID *int64          `json:"realmId,omitempty" db:"realm_id"`
	Auto    json.RawMessage `json:"auto" db:"auto"`
	Teleop  json.RawMessage `json:"teleop" db:"teleop"`
}

// CreateSchema creates a new schema
func (s *Service) CreateSchema(schema Schema) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.NamedExec(`
	INSERT
		INTO
			schemas (id, year, realm_id, auto, teleop)
		VALUES (:id, :year, :realm_id, :auto, :teleop)
	`, schema)

	if err, ok := err.(*pq.Error); ok && err.Code == pgExists {
		_ = tx.Rollback()
		return &ErrExists{msg: fmt.Sprintf("schema already exists: %v", err.Error())}
	} else if err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to insert schema")
	}

	return errors.Wrap(tx.Commit(), "unable to commit transaction")
}

// GetSchemaByID retrieves a schema given its ID
func (s *Service) GetSchemaByID(id int64) (Schema, error) {
	var schema Schema

	err := s.db.Get(&schema, "SELECT * FROM schemas WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return schema, &ErrNoResults{msg: fmt.Sprintf("schema %d does not exist", schema.ID)}
	}

	return schema, errors.Wrap(err, "unable to retrieve schema")
}

// GetSchemaByYear retrieves the schema for a given year
func (s *Service) GetSchemaByYear(year int) (Schema, error) {
	var schema Schema

	err := s.db.Get(&schema, "SELECT * FROM schemas WHERE year = $1", year)
	if err == sql.ErrNoRows {
		return schema, &ErrNoResults{msg: fmt.Sprintf("no schema for year %d exists", year)}
	}

	return schema, errors.Wrap(err, "unable to retrieve schema")
}

// GetVisibleSchemas retrieves schemas from the database frm a specific realm,
// from realms with public events, and standard FRC schemas. If the realm ID is
// nil, no private realms' schemas will be retrieved.
func (s *Service) GetVisibleSchemas(realmID *int64) ([]Schema, error) {
	schemas := []Schema{}
	var err error

	if realmID == nil {
		err = s.db.Select(&schemas, `
		WITH public_realms AS (
			SELECT id FROM realms WHERE share_reports = true
		)
		SELECT * 
			FROM schemas
				WHERE year IS NULL OR realm_id IN (SELECT id FROM public_realms)
		`)
	} else {
		err = s.db.Select(&schemas, `
		WITH public_realms AS (
			SELECT id FROM realms WHERE share_reports = true
		)
		SELECT * 
			FROM schemas
			    WHERE year IS NULL OR realm_id = $1 OR (SELECT id FROM public_realms)
		`, *realmID)
	}

	return schemas, errors.Wrap(err, "unable to retrieve schemas")
}

// GetSchemas retrieves all schemas from the database.
func (s *Service) GetSchemas() ([]Schema, error) {
	schemas := []Schema{}
	err := s.db.Select(&schemas, "SELECT * FROM schemas")
	return schemas, errors.Wrap(err, "unable to retrieve schemas")
}
