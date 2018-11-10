package store

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

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
		return fmt.Errorf("got incorrect type for jsonb")
	}

	return json.Unmarshal(bytes, r)
}

// User holds information about a user such as their id, username, and hashed
// password.
type User struct {
	ID             int64          `json:"id" db:"id"`
	Username       string         `json:"username" db:"username"`
	HashedPassword string         `json:"-" db:"hashed_password"`
	Realm          string         `json:"realm" db:"realm"`
	FirstName      string         `json:"firstName" db:"first_name"`
	LastName       string         `json:"lastName" db:"last_name"`
	Roles          Roles          `json:"roles" db:"roles"`
	Stars          pq.StringArray `json:"stars" db:"stars"`
}

// PatchUser is like User but with all nullable fields (besides id) for patching.
type PatchUser struct {
	ID             int64          `json:"id" db:"id"`
	Username       *string        `json:"username" db:"username"`
	HashedPassword *string        `json:"-" db:"hashed_password"`
	FirstName      *string        `json:"firstName" db:"first_name"`
	LastName       *string        `json:"lastName" db:"last_name"`
	Roles          *Roles         `json:"roles" db:"roles"`
	Stars          pq.StringArray `json:"stars"`
}

// GetUserByUsername retrieves a user from the database by username. It does not
// retrieve the users stars.
func (s *Service) GetUserByUsername(username string) (User, error) {
	var u User

	err := s.db.Get(&u, "SELECT * FROM users WHERE username = $1", username)
	if err == sql.ErrNoRows {
		return u, &ErrNoResults{msg: fmt.Sprintf("user %d does not exist", u.ID)}
	}

	return u, errors.Wrap(err, "unable to select user")
}

// CreateUser creates a given user.
func (s *Service) CreateUser(u User) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	_, err = tx.NamedExec(`
	INSERT
		INTO
			users (username, hashed_password, realm, first_name, last_name, roles)
		VALUES (:username, :hashed_password, :realm, :first_name, :last_name, :roles)
	`, u)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			_ = tx.Rollback()
			if err.Code == pgExists {
				return &ErrExists{msg: fmt.Sprintf("username %s already exists", u.Username)}
			}
			if err.Code == pgFKeyViolation {
				return &ErrFKeyViolation{msg: fmt.Sprintf("user fk violation on realm %s: %v", u.Realm, err)}
			}
			return err
		}
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to insert user")
	}

	starsStmt, err := tx.Prepare("INSERT INTO stars (user_id, event_key) VALUES ($1, $2)")
	if err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to prepare stars insert statement")
	}

	for _, star := range u.Stars {
		if _, err := starsStmt.Exec(u.ID, star); err != nil {
			_ = tx.Rollback()
			if err, ok := err.(*pq.Error); ok && err.Code == pgFKeyViolation {
				return &ErrFKeyViolation{msg: fmt.Sprintf("user stars event key fk violation: %v", err)}
			}
			return errors.Wrap(err, "unable to insert star for user")
		}
	}

	return errors.Wrap(tx.Commit(), "unable to commit transaction")
}

// GetUsers retrieves all users.
func (s *Service) GetUsers() ([]User, error) {
	users := []User{}

	err := s.db.Select(&users, `
	SELECT
		id,
		username,
		hashed_password,
		realm,
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
func (s *Service) GetUsersByRealm(realm string) ([]User, error) {
	users := []User{}

	err := s.db.Select(&users, `
	SELECT
		id,
		username,
		hashed_password,
		realm,
		first_name,
		last_name,
		roles,
		array_remove(array_agg(stars.event_key), NULL) AS stars
	FROM users
	LEFT JOIN
		stars
	ON
		stars.user_id = users.id
	WHERE realm = $1
	GROUP BY users.id
	`, realm)

	return users, errors.Wrap(err, "unable to fetch users")
}

// GetUserByID retrieves a user from the database by id.
func (s *Service) GetUserByID(id int64) (User, error) {
	var u User

	err := s.db.Get(&u, `
	SELECT
		id,
		username,
		hashed_password,
		realm,
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
		return u, &ErrNoResults{msg: fmt.Sprintf("user %d does not exist", u.ID)}
	}

	return u, errors.Wrap(err, "unable to select user")
}

// PatchUser updates a user by their ID.
func (s *Service) PatchUser(pu PatchUser) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	result, err := tx.NamedExec(`
	UPDATE users
	    SET
		    username = COALESCE(:username, username),
		    hashed_password = COALESCE(:hashed_password, hashed_password),
		    first_name = COALESCE(:first_name, first_name),
		    last_name = COALESCE(:last_name, last_name),
		    roles = COALESCE(:roles, roles)
	    WHERE
		    id = :id
	`, pu)
	if err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to patch user")
	}

	if count, err := result.RowsAffected(); err != nil || count == 0 {
		_ = tx.Rollback()
		return &ErrNoResults{msg: fmt.Sprintf("user ID %d not found", pu.ID)}
	}

	if pu.Stars != nil {
		if _, err := tx.Exec("DELETE FROM stars WHERE user_id = $1", pu.ID); err != nil {
			_ = tx.Rollback()
			return errors.Wrap(err, "unable to remove user stars")
		}

		starsStmt, err := tx.Prepare("INSERT INTO stars (user_id, event_key) VALUES ($1, $2)")
		if err != nil {
			_ = tx.Rollback()
			return errors.Wrap(err, "unable to prepare stars insert statement")
		}

		for _, star := range pu.Stars {
			if _, err := starsStmt.Exec(pu.ID, star); err != nil {
				_ = tx.Rollback()
				if err, ok := err.(*pq.Error); ok && err.Code == pgFKeyViolation {
					return &ErrFKeyViolation{msg: fmt.Sprintf("user stars event key fk violation: %v", err)}
				}
				return errors.Wrap(err, "unable to insert star for user")
			}
		}
	}

	return errors.Wrap(tx.Commit(), "unable to patch user")
}

// DeleteUser deletes a specific user from the database.
func (s *Service) DeleteUser(id int64) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	if _, err := tx.Exec(`
	    DELETE FROM stars
	        WHERE user_id = $1
	`, id); err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to delete user's stars")
	}

	if _, err := tx.Exec(`
	    DELETE FROM users
		    WHERE id = $1
	`, id); err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to delete user")
	}
	return errors.Wrap(tx.Commit(), "unable to delete user")
}
