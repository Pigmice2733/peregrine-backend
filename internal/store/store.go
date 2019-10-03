package store

import (
	"context"
	"fmt"

	"errors"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Register lib/pq PostreSQL driver
	"github.com/sirupsen/logrus"
)

// ErrNoResults indicates that no data matching the query was found.
type ErrNoResults struct {
	error
}

// Is returns whether the target is an ErrNoResults.
func (err ErrNoResults) Is(target error) bool {
	_, ok := target.(ErrNoResults)
	return ok
}

// ErrExists is returned if a unique record already exists.
type ErrExists struct {
	error
}

// Is returns whether the target is an ErrExists.
func (err ErrExists) Is(target error) bool {
	_, ok := target.(ErrExists)
	return ok
}

// ErrFKeyViolation is returned if inserting a record causes a foreign key violation.
type ErrFKeyViolation struct {
	error
}

// Is returns whether the target is an ErrFKeyViolation.
func (err ErrFKeyViolation) Is(target error) bool {
	_, ok := target.(ErrFKeyViolation)
	return ok
}

const pgExists = "23505"
const pgFKeyViolation = "23503"

// Service provides methods for storing data in a PostgreSQL database.
type Service struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// New creates a new store service from a dataSourceName. The logger is used to
// log errors that would not otherwise be returned such as issues rolling back
// transactions. The context passed is used for pinging the database.
func New(ctx context.Context, dsn string, logger *logrus.Logger) (*Service, error) {
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	s := &Service{db: db, logger: logger}
	return s, s.Ping(ctx)
}

// Ping pings the underlying postgresql database. You would think we would call
// db.Ping() here, but that doesn't actually Ping the database because reasons.
func (s *Service) Ping(ctx context.Context) error {
	if s.db != nil {
		return s.db.QueryRowContext(ctx, "SELECT 1").Scan(new(bool))
	}

	return errors.New("not connected to postgresql")
}

// Close closes the underlying postgresql database.
func (s *Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}

	return nil
}

func (s *Service) logErr(err error) {
	if err != nil {
		s.logger.Error(err)
	}
}

// DoTransaction opens a SQL transaction and calls txWrapper with the transaction. If the txWrapper
// return an error, the transaction will be rolled back.
func (s *Service) DoTransaction(ctx context.Context, txWrapper func(*sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}

	if err := txWrapper(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && ctx.Err() != context.Canceled {
			s.logErr(fmt.Errorf("unable to rollback transaction: %w", err))
		}

		return fmt.Errorf("error in transaction wrapper: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit transaction: %w", err)
	}

	return nil
}
