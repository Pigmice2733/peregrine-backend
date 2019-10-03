package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Schema describes the statistics that reports should include
type Schema struct {
	ID      int64        `json:"id" db:"id"`
	Year    *int64       `json:"year,omitempty" db:"year"`
	RealmID *int64       `json:"realmId,omitempty" db:"realm_id"`
	Schema  SchemaFields `json:"schema" db:"schema"`
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
	ReportReference string            `json:"reportReference,omitempty"`
	TBAReference    string            `json:"tbaReference,omitempty"`
	Sum             []FieldDescriptor `json:"sum,omitempty"`
	AnyOf           []EqualExpression `json:"anyOf,omitempty"`

	Hide   bool   `json:"hide,omitempty"`
	Type   string `json:"type,omitempty"`
	Period string `json:"period,omitempty"`
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
	return s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.NamedExecContext(ctx, `
		INSERT
			INTO
				schemas (year, realm_id, schema)
			VALUES (:year, :realm_id, :schema)
		`, schema)

		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == pgExists {
			return &ErrExists{fmt.Errorf("schema already exists: %v", err.Error())}
		} else if err != nil {
			return fmt.Errorf("unable to insert schema: %w", err)
		}

		return nil
	})
}

// GetSchemaByID retrieves a schema given its ID
func (s *Service) GetSchemaByID(ctx context.Context, id int64) (Schema, error) {
	var schema Schema

	err := s.db.GetContext(ctx, &schema, "SELECT * FROM schemas WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return schema, ErrNoResults{fmt.Errorf("schema %d does not exist", schema.ID)}
	} else if err != nil {
		return schema, fmt.Errorf("unable to retrieve schema: %w", err)
	}

	return schema, nil
}

// GetSchemaByYear retrieves the schema for a given year
func (s *Service) GetSchemaByYear(ctx context.Context, year int) (Schema, error) {
	var schema Schema

	err := s.db.GetContext(ctx, &schema, "SELECT * FROM schemas WHERE year = $1", year)
	if err == sql.ErrNoRows {
		return schema, ErrNoResults{fmt.Errorf("no schema for year %d exists", year)}
	} else if err != nil {
		return schema, fmt.Errorf("unable to retrieve schema: %w", err)
	}

	return schema, nil
}

// GetSchemasForRealm retrieves schemas from the database frm a specific realm,
// from realms with public events, and standard FRC schemas. If the realm ID is
// nil, no private realms' schemas will be retrieved.
func (s *Service) GetSchemasForRealm(ctx context.Context, realmID *int64) ([]Schema, error) {
	schemas := []Schema{}

	err := s.db.SelectContext(ctx, &schemas, `
	SELECT schemas.*
	FROM schemas
	LEFT JOIN realms
		ON realms.id = schemas.realm_id
	WHERE
		schemas.year IS NULL OR
		realms.id = NULL OR
		(realms.share_reports = true OR realms.id = $1)
	`, realmID)
	if err != nil {
		return schemas, fmt.Errorf("unable to retrieve schemas: %w", err)
	}

	return schemas, nil
}
