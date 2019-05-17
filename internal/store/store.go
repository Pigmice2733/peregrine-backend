package store

import (
	"context"

	"github.com/pkg/errors"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Register lib/pq PostreSQL driver
	"github.com/sirupsen/logrus"
)

// ErrNoResults indicates that no data matching the query was found.
type ErrNoResults struct {
	error
}

// ErrExists is returned if a unique record already exists.
type ErrExists struct {
	error
}

// ErrFKeyViolation is returned if inserting a record causes a foreign key violation.
type ErrFKeyViolation struct {
	error
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

func (s *Service) doTransaction(ctx context.Context, txWrapper func(*sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	if err := txWrapper(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			s.logErr(errors.Wrap(err, "unable to rollback transaction"))
		}

		return errors.Wrap(err, "error in transaction wrapper")
	}

	return errors.Wrap(tx.Commit(), "unable to commit transaction")
}
