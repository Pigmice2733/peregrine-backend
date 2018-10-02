package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	validator "gopkg.in/go-playground/validator.v9"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	jwt "github.com/dgrijalva/jwt-go"
)

type contextKey string

const (
	keyRolesContext   contextKey = "pigmice_roles"
	keySubjectContext contextKey = "pigmice_subject"
)

type baseUser struct {
	Username string `json:"username" validate:"gte=4,lte=32"`
	Password string `json:"password" validate:"gte=8,lte=128"`
}

func (s *Server) authenticateHandler() http.HandlerFunc {
	type requestUser baseUser

	return func(w http.ResponseWriter, r *http.Request) {
		var ru requestUser
		if err := json.NewDecoder(r.Body).Decode(&ru); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		if err := validator.New().Struct(ru); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		user, err := s.store.GetUser(ru.Username)
		if err != nil {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}

		if !contains(user.Roles, verifiedRole) {
			ihttp.Respond(w, fmt.Errorf("user has not yet been verified"), http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(ru.Password))
		if err == bcrypt.ErrMismatchedHashAndPassword {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		} else if err != nil {
			s.logger.Printf("Error: comparing hash and password: %v\n", err)
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ss, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour * 8).Unix(),
				Subject:   strconv.FormatInt(user.ID, 10),
			},
			Roles: user.Roles,
		}).SignedString(s.jwtSecret)
		if err != nil {
			s.logger.Printf("Error: generating jwt signed string: %v\n", err)
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, map[string]string{"jwt": ss}, http.StatusOK)
	}
}

func contains(arr []string, a string) bool {
	for _, v := range arr {
		if v == a {
			return true
		}
	}

	return false
}

func getRoles(r *http.Request) []string {
	contextRoles := r.Context().Value(keyRolesContext)
	if contextRoles == nil {
		return []string{}
	}

	roles, ok := contextRoles.([]string)
	if !ok {
		return []string{}
	}

	return roles
}

func (s *Server) createUserHandler() http.HandlerFunc {
	type requestUser struct {
		baseUser
		FirstName string   `json:"firstName" validate:"required"`
		LastName  string   `json:"lastName" validate:"required"`
		Roles     []string `json:"roles"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var ru requestUser
		if err := json.NewDecoder(r.Body).Decode(&ru); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		// If the creator user isn't an admin, reset their roles
		if !contains(getRoles(r), adminRole) {
			ru.Roles = make([]string, 0)
		}

		if err := validator.New().Struct(ru); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		u := store.User{Username: ru.Username, Roles: ru.Roles}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(ru.Password), bcrypt.DefaultCost)
		if err != nil {
			s.logger.Printf("Error: hashing user password: %v\n", err)
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		u.HashedPassword = string(hashedPassword)

		err = s.store.CreateUser(u)
		if err == store.ErrExists {
			ihttp.Respond(w, err, http.StatusConflict)
			return
		} else if err != nil {
			s.logger.Printf("Error: creating new user: %v\n", err)
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}
	}
}
