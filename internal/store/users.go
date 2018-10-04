package store

import (
	"fmt"

	"github.com/lib/pq"
)

// ErrExists is returned if a unique record already exists.
var ErrExists = fmt.Errorf("a record already exists")

const pgExists = "23505"

// User holds information about a user such as their id, username, and hashed
// password.
type User struct {
	ID             int64    `json:"id"`
	Username       string   `json:"username"`
	HashedPassword string   `json:"hashedPassword"`
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	Roles          []string `json:"roles"`
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
		pq.Array(&u.Roles),
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
	`, u.Username, u.HashedPassword, u.FirstName, u.LastName, pq.Array(u.Roles))

	if err, ok := err.(*pq.Error); ok {
		if err.Code == pgExists {
			return ErrExists
		}
	}
	return err
}
