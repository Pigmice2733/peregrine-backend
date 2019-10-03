package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Roles holds information about a users roles and permissions such as whether
// they are an administrator.
type Roles struct {
	IsSuperAdmin bool `json:"isSuperAdmin" yaml:"isSuperAdmin"`
	IsAdmin      bool `json:"isAdmin" yaml:"isAdmin"`
	IsVerified   bool `json:"isVerified" yaml:"isVerified"`
}

// Value is provided for returning the value of Roles as marshalled JSON for
// PostgreSQL's JSONB type.
func (r Roles) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// Scan is provided for scanning data from PostgreSQL's JSONB type into Roles.
func (r *Roles) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return errors.New("got incorrect type for jsonb")
	}

	return json.Unmarshal(bytes, r)
}

// User holds information about a user such as their id, username, and hashed
// password.
type User struct {
	ID              int64          `json:"id" db:"id"`
	Username        string         `json:"username" db:"username"`
	HashedPassword  string         `json:"-" db:"hashed_password"`
	PasswordChanged time.Time      `json:"-" db:"password_changed"`
	RealmID         int64          `json:"realmId" db:"realm_id"`
	FirstName       string         `json:"firstName" db:"first_name"`
	LastName        string         `json:"lastName" db:"last_name"`
	Roles           Roles          `json:"roles" db:"roles"`
	Stars           pq.StringArray `json:"stars" db:"stars"`
}

// PatchUser is like User but with all nullable fields (besides id and realmID) for patching.
type PatchUser struct {
	ID              int64          `json:"id" db:"id"`
	Username        *string        `json:"username" db:"username"`
	HashedPassword  *string        `json:"-" db:"hashed_password"`
	PasswordChanged *time.Time     `json:"-" db:"password_changed"`
	FirstName       *string        `json:"firstName" db:"first_name"`
	LastName        *string        `json:"lastName" db:"last_name"`
	Roles           *Roles         `json:"roles" db:"roles"`
	Stars           pq.StringArray `json:"stars"`
}

// GetUserByUsername retrieves a user from the database by username. It does not
// retrieve the users stars.
func (s *Service) GetUserByUsername(ctx context.Context, username string) (User, error) {
	var u User

	err := s.db.GetContext(ctx, &u, "SELECT * FROM users WHERE username = $1", username)
	if err == sql.ErrNoRows {
		return u, ErrNoResults{errors.Wrapf(err, "user %d does not exist", u.ID)}
	}

	return u, errors.Wrap(err, "unable to select user")
}

// CreateUser creates a given user.
func (s *Service) CreateUser(ctx context.Context, u User) error {
	return s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		u.PasswordChanged = time.Now()

		userStmt, err := tx.PrepareNamedContext(ctx, `
		INSERT
			INTO
				users (username, hashed_password, password_changed, realm_id, first_name, last_name, roles)
			VALUES (:username, :hashed_password, :password_changed, :realm_id, :first_name, :last_name, :roles)
			RETURNING id
		`)
		if err != nil {
			return errors.Wrap(err, "unable to prepare user insert statement")
		}

		err = userStmt.GetContext(ctx, &u.ID, u)
		if err != nil {
			if err, ok := err.(*pq.Error); ok {
				if err.Code == pgExists {
					return ErrExists{errors.Wrapf(err, "username %s already exists", u.Username)}
				}
				if err.Code == pgFKeyViolation {
					return ErrFKeyViolation{errors.Wrapf(err, "user fk violation on realm ID %d: %v", u.RealmID, err)}
				}
			}
			return errors.Wrap(err, "unable to insert user")
		}

		starsStmt, err := tx.PrepareContext(ctx, "INSERT INTO stars (user_id, event_key) VALUES ($1, $2)")
		if err != nil {
			return errors.Wrap(err, "unable to prepare stars insert statement")
		}

		for _, star := range u.Stars {
			if _, err := starsStmt.ExecContext(ctx, u.ID, star); err != nil {
				if err, ok := err.(*pq.Error); ok && err.Code == pgFKeyViolation {
					return ErrFKeyViolation{errors.Wrapf(err, "user stars event key fk violation: %v", err)}
				}
				return errors.Wrap(err, "unable to insert star for user")
			}
		}

		return nil
	})
}

// GetUsers retrieves all users.
func (s *Service) GetUsers(ctx context.Context) ([]User, error) {
	users := []User{}

	err := s.db.SelectContext(ctx, &users, `
	SELECT
		id,
		username,
		hashed_password,
		password_changed,
		realm_id,
		first_name,
		last_name,
		roles,
		array_remove(array_agg(stars.event_key), NULL) AS stars
	FROM users
	LEFT JOIN
		stars
	ON
		stars.user_id = users.id
	GROUP BY users.id
	`)

	return users, errors.Wrap(err, "unable to fetch users")
}

// GetUsersByRealm retrieves all users in a specific realm.
func (s *Service) GetUsersByRealm(ctx context.Context, realmID int64) ([]User, error) {
	users := []User{}

	err := s.db.SelectContext(ctx, &users, `
	SELECT
		id,
		username,
		hashed_password,
		password_changed,
		realm_id,
		first_name,
		last_name,
		roles,
		array_remove(array_agg(stars.event_key), NULL) AS stars
	FROM users
	LEFT JOIN
		stars
	ON
		stars.user_id = users.id
	WHERE realm_id = $1
	GROUP BY users.id
	`, realmID)

	return users, errors.Wrap(err, "unable to fetch users")
}

// GetUserByID retrieves a user from the database by id.
func (s *Service) GetUserByID(ctx context.Context, id int64) (User, error) {
	var u User

	err := s.db.GetContext(ctx, &u, `
	SELECT
		id,
		username,
		hashed_password,
		password_changed,
		realm_id,
		first_name,
		last_name,
		roles,
		array_remove(array_agg(stars.event_key), NULL) AS stars
	FROM users
	LEFT JOIN
		stars
	ON
		stars.user_id = users.id
	WHERE id = $1
	GROUP BY users.id
	`, id)
	if err == sql.ErrNoRows {
		return u, ErrNoResults{errors.Wrapf(err, "user %d does not exist", u.ID)}
	}

	return u, errors.Wrap(err, "unable to select user")
}

// PatchUser updates a user by their ID.
func (s *Service) PatchUser(ctx context.Context, pu PatchUser) error {
	return s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		if pu.HashedPassword != nil {
			now := time.Now()
			pu.PasswordChanged = &now
		}

		result, err := tx.NamedExecContext(ctx, `
		UPDATE users
			SET
				username = COALESCE(:username, username),
				hashed_password = COALESCE(:hashed_password, hashed_password),
				password_changed = COALESCE(:password_changed, password_changed),
				first_name = COALESCE(:first_name, first_name),
				last_name = COALESCE(:last_name, last_name),
				roles = COALESCE(:roles, roles)
			WHERE
				id = :id
		`, pu)
		if err != nil {
			return errors.Wrap(err, "unable to patch user")
		}

		if count, err := result.RowsAffected(); err != nil || count == 0 {
			return ErrNoResults{errors.Wrapf(err, "user ID %d not found", pu.ID)}
		}

		if pu.Stars != nil {
			if _, err := tx.ExecContext(ctx, "DELETE FROM stars WHERE user_id = $1", pu.ID); err != nil {
				return errors.Wrap(err, "unable to remove user stars")
			}

			starsStmt, err := tx.PrepareContext(ctx, "INSERT INTO stars (user_id, event_key) VALUES ($1, $2)")
			if err != nil {
				return errors.Wrap(err, "unable to prepare stars insert statement")
			}

			for _, star := range pu.Stars {
				if _, err := starsStmt.ExecContext(ctx, pu.ID, star); err != nil {
					if err, ok := err.(*pq.Error); ok && err.Code == pgFKeyViolation {
						return ErrFKeyViolation{errors.Wrapf(err, "user stars event key fk violation: %v", err)}
					}
					return errors.Wrap(err, "unable to insert star for user")
				}
			}
		}

		return nil
	})
}

// DeleteUserByID deletes a specific user from the database.
func (s *Service) DeleteUserByID(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("unable to delete user %d: %w", id, err)
	}

	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return ErrNoResults{errors.New("got 0 affected rows")}
	}

	return nil
}

// DeleteUserByIDRealm deletes a specific user from the database.
func (s *Service) DeleteUserByIDRealm(ctx context.Context, id, realmID int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1 AND realm_id = $2", id, realmID)
	if err != nil {
		return fmt.Errorf("unable to delete user %d: %w", id, err)
	}

	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return ErrNoResults{errors.New("got 0 affected rows")}
	}

	return nil
}

// CheckSimilarUsernameExists checks whether a user with (case insensitive) the
// same username exists. It returns an ErrExists if a similar user exists.
// If an id is given, it will ignore the user with that id.
func (s *Service) CheckSimilarUsernameExists(ctx context.Context, username string, id *int64) error {
	var ok bool
	var err error

	if id != nil {
		err = s.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT true FROM users WHERE lower(username) = lower($1) AND id != $2)`,
			username, id).Scan(&ok)
	} else {
		err = s.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT true FROM users WHERE lower(username) = lower($1))`,
			username).Scan(&ok)
	}

	if err != nil {
		return errors.Wrap(err, "unable to select whether user exists")
	}

	if ok {
		return ErrExists{errors.New("user with similar username exists")}
	}

	return nil
}
