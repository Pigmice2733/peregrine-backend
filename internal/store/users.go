package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
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
	ID             int64  `json:"id"`
	Username       string `json:"username"`
	HashedPassword string `json:"hashedPassword"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Roles          Roles  `json:"roles"`
}

// GetUser retrieves a user by username.
func (s *Service) GetUser(username string) (User, error) {
	var u User

	err := s.db.QueryRow(
		`SELECT 
			id, username, hashed_password, first_name, last_name, roles
			FROM
				users WHERE username = $1`,
		username,
	).Scan(
		&u.ID,
		&u.Username,
		&u.HashedPassword,
		&u.FirstName,
		&u.LastName,
		&u.Roles,
	)

	return u, err
}

// CreateUser creates a given user.
func (s *Service) CreateUser(u User) error {
	_, err := s.db.Exec(`
	INSERT
		INTO
			users (username, hashed_password, first_name, last_name, roles)
		VALUES ($1, $2, $3, $4, $5)
	`, u.Username, u.HashedPassword, u.FirstName, u.LastName, u.Roles)

	if err, ok := err.(*pq.Error); ok {
		if err.Code == pgExists {
			return ErrExists
		}
	}
	return err
}
