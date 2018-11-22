package store

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// StatDescription escribes a single statistic in a schema
type StatDescription struct {
	StatName string `json:"statName"`
	Type     string `json:"type"`
}

// SchemaSection wraps one section of the schema (auto, teleop) into a type
// that can be used with PostgreSQL.
type SchemaSection []StatDescription

// Value converts a SchemaSection into JSON for PostgreSQL's JSONB type.
func (d *SchemaSection) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// Scan converts data from PostgreSQL's JSONB type into SchemaSection.
func (d *SchemaSection) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("got incorrect type for JSONB")
	}

	return json.Unmarshal(bytes, d)
}

// Schema describes the statistics that reports should include
type Schema struct {
	ID      int64          `json:"id" db:"id"`
	Year    *int64         `json:"year,omitempty" db:"year"`
	RealmID *int64         `json:"realmId,omitempty" db:"realm_id"`
	Auto    *SchemaSection `json:"auto" db:"auto"`
	Teleop  *SchemaSection `json:"teleop" db:"teleop"`
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

// GetSchemas retrieves all schemas from the database
func (s *Service) GetSchemas() ([]Schema, error) {
	schemas := []Schema{}

	err := s.db.Select(&schemas, "SELECT * FROM schemas")

	return schemas, errors.Wrap(err, "unable to retrieve schemas")
}
