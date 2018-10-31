package store

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// ErrExists is returned if a unique record already exists.
var ErrExists = fmt.Errorf("a record already exists")

const pgExists = "23505"

// Roles holds information about a users roles and permissions such as whether
// they are an administrator.
type Roles struct {
	IsAdmin    bool `json:"isAdmin" yaml:"isAdmin"`
	IsVerified bool `json:"isVerified" yaml:"isVerified"`
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
	ID             int64    `json:"id" db:"id"`
	Username       string   `json:"username" db:"username"`
	HashedPassword string   `json:"-" db:"hashed_password"`
	FirstName      string   `json:"firstName" db:"first_name"`
	LastName       string   `json:"lastName" db:"last_name"`
	Roles          Roles    `json:"roles" db:"roles"`
	Stars          []string `json:"stars"`
}

// PatchUser is like User but with all nullable fields (besides id) for patching.
type PatchUser struct {
	ID             int64    `json:"id" db:"id"`
	Username       *string  `json:"username" db:"username"`
	HashedPassword *string  `json:"-" db:"hashed_password"`
	FirstName      *string  `json:"firstName" db:"first_name"`
	LastName       *string  `json:"lastName" db:"last_name"`
	Roles          *Roles   `json:"roles" db:"roles"`
	Stars          []string `json:"stars"`
}

// GetUserByUsername retrieves a user by username.
func (s *Service) GetUserByUsername(username string) (User, error) {
	var u User

	tx, err := s.db.Beginx()
	if err != nil {
		return u, errors.Wrap(err, "unable to begin transaction")
	}

	err = tx.Get(&u, "SELECT * FROM users WHERE username = $1", username)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		return u, ErrNoResults(err)
	} else if err != nil {
		_ = tx.Rollback()
		return u, errors.Wrap(err, "unable to select user")
	}

	if err := tx.Select(&u.Stars, "SELECT event_key FROM stars WHERE user_id = $1", u.ID); err != nil {
		_ = tx.Rollback()
		return u, errors.Wrap(err, "unable to select stars for user")
	}

	return u, errors.Wrap(tx.Commit(), "unable to commit transaction")
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
			users (username, hashed_password, first_name, last_name, roles)
		VALUES (:username, :hashed_password, :first_name, :last_name, :roles)
	`, u)
	if err, ok := err.(*pq.Error); ok {
		if err.Code == pgExists {
			_ = tx.Rollback()
			return ErrExists
		}
	} else if err != nil {
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
			return errors.Wrap(err, "unable to insert star for user")
		}
	}

	return errors.Wrap(tx.Commit(), "unable to commit transaction")
}

// GetUsers retrieves all users.
func (s *Service) GetUsers() ([]User, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, errors.Wrap(err, "unable to begin transaction")
	}

	var users []User
	if err := tx.Select(&users, "SELECT * FROM users"); err != nil {
		_ = tx.Rollback()
		return users, errors.Wrap(err, "unable to select users")
	}

	for i, user := range users {
		if err := tx.Select(&user.Stars, "SELECT event_key FROM stars WHERE user_id = $1", user.ID); err != nil {
			_ = tx.Rollback()
			return users, errors.Wrap(err, "unable to select stars for user")
		}

		users[i] = user
	}

	return users, errors.Wrap(tx.Commit(), "unable to commit transaction")
}

// GetUserByID retrieves a user from the database by id.
func (s *Service) GetUserByID(id int64) (User, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return User{}, errors.Wrap(err, "unable to begin transaction")
	}

	var u User

	if err := tx.Get(&u, "SELECT * FROM users WHERE id = $1", id); err != nil {
		_ = tx.Rollback()
		return u, errors.Wrap(err, "unable to select users")
	}

	if err := tx.Select(&u.Stars, "SELECT event_key FROM stars WHERE user_id = $1", id); err != nil {
		_ = tx.Rollback()
		return u, errors.Wrap(err, "unable to select stars for user")
	}

	return u, errors.Wrap(tx.Commit(), "unable to commit transaction")
}

// PatchUser updates a user by their ID.
func (s *Service) PatchUser(pu PatchUser) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "unable to begin transaction")
	}

	if _, err := tx.NamedExec(`
	UPDATE users
	SET
		username = COALESCE(:username, username),
		hashed_password = COALESCE(:hashed_password, hashed_password),
		first_name = COALESCE(:first_name, first_name),
		last_name = COALESCE(:last_name, last_name),
		roles = COALESCE(:roles, roles)
	WHERE
		id = :id
	`, pu); err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "unable to patch user")
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
				return errors.Wrap(err, "unable to insert star for user")
			}
		}
	}

	return errors.Wrap(tx.Commit(), "unable to patch user")
}
