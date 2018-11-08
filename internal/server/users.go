package server

import (
	"encoding/json"
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

type requestUser struct {
	baseUser
	Realm     string      `json:"realm" validate:"required"`
	FirstName string      `json:"firstName" validate:"required"`
	LastName  string      `json:"lastName" validate:"required"`
	Roles     store.Roles `json:"roles"`
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
		if err == store.ErrNoResults {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		} else if err != nil {
			go s.logger.WithError(err).Error("retrieving user from database")
			ihttp.Error(w, http.StatusInternalServerError)
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
			Realm: user.Realm,
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
	return func(w http.ResponseWriter, r *http.Request) {
		var ru requestUser
		if err := json.NewDecoder(r.Body).Decode(&ru); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		// If the creator user isn't an admin, reset their roles
		roles := ihttp.GetRoles(r)
		if !roles.IsAdmin && !roles.IsSuperAdmin {
			ru.Roles = store.Roles{}
		}

		// Only super-admins can create super-admins
		if !roles.IsSuperAdmin {
			ru.Roles.IsSuperAdmin = false
		}

		// Only super-admins can create verified users in realms other than their own.
		if !roles.IsSuperAdmin {
			if ihttp.GetRealm(r) != ru.Realm {
				ru.Roles.IsVerified = false
				ru.Roles.IsAdmin = false
			}
		}

		if err := validator.New().Struct(ru); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		u := store.User{Username: ru.Username, Realm: ru.Realm, Roles: ru.Roles, FirstName: ru.FirstName, LastName: ru.LastName}

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
		} else if err == store.ErrFKeyViolation {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
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
		roles := ihttp.GetRoles(r)

		var users []store.User
		var err error

		if roles.IsSuperAdmin {
			users, err = s.store.GetUsers()
		} else {
			users, err = s.store.GetUsersByRealm(ihttp.GetRealm(r))
		}

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
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}
		roles := ihttp.GetRoles(r)

		// If the user is not an admin and their ID does not equal the ID they
		// are trying to get, they are forbidden
		if !roles.IsAdmin && !roles.IsSuperAdmin && sub != id {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		user, err := s.store.GetUserByID(id)
		if err == store.ErrNoResults {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			go s.logger.WithError(err).Error("getting user by id")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		if roles.IsSuperAdmin && sub != id {
			if ihttp.GetRealm(r) != user.Realm {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
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
		targetID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		subjectID, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}
		roles := ihttp.GetRoles(r)

		if targetID != subjectID && !roles.IsAdmin && !roles.IsSuperAdmin {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		var ru patchUser
		if err := json.NewDecoder(r.Body).Decode(&ru); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		// Admins can only patch users in the same realm
		if targetID != subjectID && !roles.IsSuperAdmin {
			targetUser, err := s.store.GetUserByID(targetID)
			if err != nil {
				go s.logger.WithError(err).Error("getting user")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}
			if ihttp.GetRealm(r) != targetUser.Realm {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		// If the creator user isn't an admin, reset roles
		if !roles.IsAdmin && !roles.IsSuperAdmin {
			ru.Roles = nil
		}

		// Only super-admins can create super-admins
		if ru.Roles != nil && !roles.IsSuperAdmin {
			ru.Roles.IsSuperAdmin = false
		}

		if err := validator.New().Struct(ru); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		u := store.PatchUser{ID: targetID, Username: ru.Username, Roles: ru.Roles, FirstName: ru.FirstName, LastName: ru.LastName, Stars: ru.Stars}

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
		if err == store.ErrNoResults {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err == store.ErrFKeyViolation {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		} else if err != nil {
			go s.logger.WithError(err).Error("patching user")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, nil, http.StatusNoContent)
	}
}

func (s *Server) deleteUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		requestID, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}
		roles := ihttp.GetRoles(r)

		if id != requestID && !roles.IsAdmin && !roles.IsSuperAdmin {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		// Admins can only delete users in the same realm
		if id != requestID && !roles.IsSuperAdmin {
			targetUser, err := s.store.GetUserByID(id)
			if err != nil {
				go s.logger.WithError(err).Error("getting user")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}
			if ihttp.GetRealm(r) != targetUser.Realm {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		err = s.store.DeleteUser(id)
		if err != nil {
			go s.logger.WithError(err).Error("deleting user")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, nil, http.StatusNoContent)
	}
}

// func (s *Server) getUserRealm(r *http.Request) (string, error) {
// 	ID, err := ihttp.GetSubject(r)
// 	if err != nil {
// 		return "", err
// 	}
// 	user, err := s.store.GetUserByID(ID)
// 	return user.Realm, err
// }
