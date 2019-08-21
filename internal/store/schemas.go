package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Schema describes the statistics that reports should include
type Schema struct {
	ID      int64        `json:"id" db:"id"`
	Year    *int64       `json:"year,omitempty" db:"year"`
	RealmID *int64       `json:"realmId,omitempty" db:"realm_id"`
	Schema  SchemaFields `json:"auto" db:"auto"`
}

// FieldDescriptor defines properties of a schema field that aren't related to how it should be
// summarized, but just information about the field (name, period, type).
type FieldDescriptor struct {
	Name string `json:"name"`
}

// SchemaField is a singular schema field. Only specify one of: ReportReference, TBAReference,
// Sum, or AnyOf.
type SchemaField struct {
	FieldDescriptor
	ReportReference string            `json:"reportReference"`
	TBAReference    string            `json:"tbaReference"`
	Sum             []FieldDescriptor `json:"sum"`
	AnyOf           []EqualExpression `json:"anyOf"`

	Hide   bool   `json:"hide"`
	Type   string `json:"type"`
	Period string `json:"period"`
}

// EqualExpression defines a reference that should equal some JSON value (float64, number,
// string).
type EqualExpression struct {
	FieldDescriptor
	Equals interface{} `json:"equals"`
}

// SchemaFields holds multiple SchemaFields for storing in one DB column
type SchemaFields []SchemaField

// Value implements driver.Valuer to return JSON for the DB from StatDescription.
func (sd SchemaFields) Value() (driver.Value, error) { return json.Marshal(sd) }

// Scan implements sql.Scanner to scan JSON from the DB into SchemaFields.
func (sd *SchemaFields) Scan(src interface{}) error {
	j, ok := src.([]byte)
	if !ok {
		return errors.New("got invalid type for SchemaFields")
	}

	return json.Unmarshal(j, sd)
}

// CreateSchema creates a new schema
func (s *Service) CreateSchema(ctx context.Context, schema Schema) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.NamedExecContext(ctx, `
	INSERT
		INTO
			schemas (year, realm_id, schema)
		VALUES (:year, :realm_id, :schema)
	`, schema)

	if err, ok := err.(*pq.Error); ok && err.Code == pgExists {
		_ = tx.Rollback()
		return &ErrExists{fmt.Errorf("schema already exists: %v", err.Error())}
	} else if err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to insert schema")
	}

	return errors.Wrap(tx.Commit(), "unable to commit transaction")
}

// GetSchemaByID retrieves a schema given its ID
func (s *Service) GetSchemaByID(ctx context.Context, id int64) (Schema, error) {
	var schema Schema

	err := s.db.GetContext(ctx, &schema, "SELECT * FROM schemas WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return schema, ErrNoResults{fmt.Errorf("schema %d does not exist", schema.ID)}
	}

	return schema, errors.Wrap(err, "unable to retrieve schema")
}

// GetSchemaByYear retrieves the schema for a given year
func (s *Service) GetSchemaByYear(ctx context.Context, year int) (Schema, error) {
	var schema Schema

	err := s.db.GetContext(ctx, &schema, "SELECT * FROM schemas WHERE year = $1", year)
	if err == sql.ErrNoRows {
		return schema, ErrNoResults{fmt.Errorf("no schema for year %d exists", year)}
	}

	return schema, errors.Wrap(err, "unable to retrieve schema")
}

// GetVisibleSchemas retrieves schemas from the database frm a specific realm,
// from realms with public events, and standard FRC schemas. If the realm ID is
// nil, no private realms' schemas will be retrieved.
func (s *Service) GetVisibleSchemas(ctx context.Context, realmID *int64) ([]Schema, error) {
	schemas := []Schema{}
	var err error

	if realmID == nil {
		err = s.db.SelectContext(ctx, &schemas, `
		WITH public_realms AS (
			SELECT id FROM realms WHERE share_reports = true
		)
		SELECT * 
			FROM schemas
				WHERE year IS NULL OR realm_id IN (SELECT id FROM public_realms)
		`)
	} else {
		err = s.db.SelectContext(ctx, &schemas, `
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
func (s *Service) GetSchemas(ctx context.Context) ([]Schema, error) {
	schemas := []Schema{}
	err := s.db.SelectContext(ctx, &schemas, "SELECT * FROM schemas")
	return schemas, errors.Wrap(err, "unable to retrieve schemas")
}
