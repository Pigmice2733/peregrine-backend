package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	validator "gopkg.in/go-playground/validator.v9"
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

		user, err := s.store.GetUserByUsername(ru.Username)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		} else if err != nil {
			go s.logger.WithError(err).Error("retrieving user from database")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		if !user.Roles.IsVerified {
			ihttp.Respond(w, fmt.Errorf("user has not yet been verified"), http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(ru.Password))
		if err == bcrypt.ErrMismatchedHashAndPassword {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		} else if err != nil {
			go s.logger.WithError(err).Error("comparing user hash and password")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ss, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &ihttp.Claims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour * 8).Unix(),
				Subject:   strconv.FormatInt(user.ID, 10),
			},
			Roles: user.Roles,
		}).SignedString(s.jwtSecret)
		if err != nil {
			go s.logger.WithError(err).Error("generating jwt signed string")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, map[string]string{"jwt": ss}, http.StatusOK)
	}
}

func (s *Server) createUserHandler() http.HandlerFunc {
	type requestUser struct {
		baseUser
		FirstName string      `json:"firstName" validate:"required"`
		LastName  string      `json:"lastName" validate:"required"`
		Roles     store.Roles `json:"roles"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var ru requestUser
		if err := json.NewDecoder(r.Body).Decode(&ru); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		// If the creator user isn't an admin, reset their roles
		if !ihttp.GetRoles(r).IsAdmin {
			ru.Roles = store.Roles{}
		}

		if err := validator.New().Struct(ru); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		u := store.User{Username: ru.Username, Roles: ru.Roles, FirstName: ru.FirstName, LastName: ru.LastName}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(ru.Password), bcrypt.DefaultCost)
		if err != nil {
			go s.logger.WithError(err).Error("hashing user password")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		u.HashedPassword = string(hashedPassword)

		err = s.store.CreateUser(u)
		if err == store.ErrExists {
			ihttp.Respond(w, err, http.StatusConflict)
			return
		} else if err != nil {
			go s.logger.WithError(err).Error("creating new user")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, nil, http.StatusCreated)
	}
}

func (s *Server) getUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := s.store.GetUsers()
		if err != nil {
			go s.logger.WithError(err).Error("getting users")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, users, http.StatusOK)
	}
}

func (s *Server) getUserByIDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		sub, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		// if the user is not an admin and their id does not equal the id they
		// are trying to get, they are forbidden
		if !ihttp.GetRoles(r).IsAdmin && sub != id {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		user, err := s.store.GetUserByID(id)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			go s.logger.WithError(err).Error("getting user by id")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, user, http.StatusOK)
	}
}

func (s *Server) patchUserHandler() http.HandlerFunc {
	type patchUser struct {
		Username  *string      `json:"username" validate:"omitempty,gte=4,lte=32"`
		Password  *string      `json:"password" validate:"omitempty,gte=8,lte=128"`
		FirstName *string      `json:"firstName" validate:"omitempty,gte=0"`
		LastName  *string      `json:"lastName" validate:"omitempty,gte=0"`
		Roles     *store.Roles `json:"roles"`
		Stars     []string     `json:"stars"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		creatorIsAdmin := ihttp.GetRoles(r).IsAdmin

		creatorID, err := ihttp.GetSubject(r)
		if err != nil || (id != creatorID && !creatorIsAdmin) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		var ru patchUser
		if err := json.NewDecoder(r.Body).Decode(&ru); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		// If the creator user isn't an admin, reset roles
		if !creatorIsAdmin {
			ru.Roles = nil
		}

		if err := validator.New().Struct(ru); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		u := store.PatchUser{ID: id, Username: ru.Username, Roles: ru.Roles, FirstName: ru.FirstName, LastName: ru.LastName, Stars: ru.Stars}

		if ru.Password != nil {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*ru.Password), bcrypt.DefaultCost)
			if err != nil {
				go s.logger.WithError(err).Error("hashing user password")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}

			hashedPasswordString := string(hashedPassword)
			u.HashedPassword = &hashedPasswordString
		}

		err = s.store.PatchUser(u)
		if err == store.ErrExists {
			ihttp.Respond(w, err, http.StatusConflict)
			return
		} else if err != nil {
			go s.logger.WithError(err).Error("patching user")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, nil, http.StatusNoContent)
	}
}
