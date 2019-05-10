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
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	validator "gopkg.in/go-playground/validator.v9"
)

type baseUser struct {
	Username string `json:"username" validate:"gte=4,lte=32,alphanum"`
	Password string `json:"password" validate:"gte=8,lte=128"`
}

type requestUser struct {
	baseUser
	RealmID   int64       `json:"realmId" validate:"required"`
	FirstName string      `json:"firstName" validate:"required"`
	LastName  string      `json:"lastName" validate:"required"`
	Roles     store.Roles `json:"roles"`
	Stars     []string    `json:"stars"`
}

const (
	accessTokenDuration  = time.Hour * 24         // 1 day
	refreshTokenDuration = time.Hour * 24 * 7 * 4 // 4 weeks
	bcryptCost           = 12                     // ~236ms per hash on my i7-8550U
)

func generateAccessToken(user store.User, secret string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, &ihttp.Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(accessTokenDuration).Unix(),
			Subject:   strconv.FormatInt(user.ID, 10),
		},
		Roles:   user.Roles,
		RealmID: user.RealmID,
	}).SignedString([]byte(secret))
}

func (s *Server) authenticateHandler() http.HandlerFunc {
	type authenticateResponse struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var ru baseUser
		if err := json.NewDecoder(r.Body).Decode(&ru); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		if err := validator.New().Struct(ru); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		user, err := s.Store.GetUserByUsername(r.Context(), ru.Username)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		} else if err != nil {
			go s.Logger.WithError(err).Error("retrieving user from database")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(ru.Password))
		if err == bcrypt.ErrMismatchedHashAndPassword {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		} else if err != nil {
			go s.Logger.WithError(err).Error("comparing user hash and password")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		accessToken, err := generateAccessToken(user, s.JWTSecret)
		if err != nil {
			go s.Logger.WithError(err).Error("generating jwt access token signed string")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &ihttp.RefreshClaims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(refreshTokenDuration).Unix(),
				Subject:   strconv.FormatInt(user.ID, 10),
			},
			PasswordChanged: user.PasswordChanged,
		}).SignedString([]byte(s.JWTSecret))
		if err != nil {
			go s.Logger.WithError(err).Error("generating jwt refresh token signed string")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, authenticateResponse{accessToken, refreshToken}, http.StatusOK)
	}
}

func (s *Server) refreshHandler() http.HandlerFunc {
	type refreshRequest struct {
		RefreshToken string `json:"refreshToken"`
	}

	type refreshResponse struct {
		AccessToken string `json:"accessToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var rr refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&rr); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		token, err := jwt.ParseWithClaims(rr.RefreshToken, &ihttp.RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(s.JWTSecret), nil
		})
		if err != nil {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*ihttp.RefreshClaims)
		if !ok {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		user, err := s.Store.GetUserByID(r.Context(), userID)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusUnauthorized)
			return
		} else if err != nil {
			go s.Logger.WithError(err).Error("retrieving user from database")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		// user password has been updated since refresh token was issued
		if user.PasswordChanged != claims.PasswordChanged {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		accessToken, err := generateAccessToken(user, s.JWTSecret)
		if err != nil {
			go s.Logger.WithError(err).Error("generating jwt access token signed string")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, refreshResponse{accessToken}, http.StatusOK)
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
			if id, err := ihttp.GetRealmID(r); err != nil || id != ru.RealmID {
				ru.Roles.IsVerified = false
				ru.Roles.IsAdmin = false
			}
		}

		if err := validator.New().Struct(ru); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		err := s.Store.CheckSimilarUsernameExists(r.Context(), ru.Username, nil)
		if _, ok := errors.Cause(err).(store.ErrExists); ok {
			ihttp.Error(w, http.StatusConflict)
			return
		} else if err != nil {
			go s.Logger.WithError(err).Error("checking whether similar user exists")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		u := store.User{Username: ru.Username, RealmID: ru.RealmID, Roles: ru.Roles, Stars: ru.Stars, FirstName: ru.FirstName, LastName: ru.LastName}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(ru.Password), bcryptCost)
		if err != nil {
			go s.Logger.WithError(err).Error("hashing user password")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		u.HashedPassword = string(hashedPassword)

		err = s.Store.CreateUser(r.Context(), u)

		if err != nil {
			switch errors.Cause(err).(type) {
			case store.ErrExists:
				ihttp.Error(w, http.StatusConflict)
			case store.ErrFKeyViolation:
				ihttp.Error(w, http.StatusUnprocessableEntity)
			default:
				go s.Logger.WithError(err).Error("creating new user")
				ihttp.Error(w, http.StatusInternalServerError)
			}

			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func (s *Server) getUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roles := ihttp.GetRoles(r)

		var users []store.User
		var err error

		if roles.IsSuperAdmin {
			users, err = s.Store.GetUsers(r.Context())
		} else {
			var realmID int64
			realmID, err = ihttp.GetRealmID(r)
			if err != nil {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
			users, err = s.Store.GetUsersByRealm(r.Context(), realmID)
		}

		if err != nil {
			go s.Logger.WithError(err).Error("getting users")
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
		roles := ihttp.GetRoles(r)

		// If the user is not an admin and their ID does not equal the ID they
		// are trying to get, they are forbidden
		if !roles.IsAdmin && !roles.IsSuperAdmin && sub != id {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		user, err := s.Store.GetUserByID(r.Context(), id)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			go s.Logger.WithError(err).Error("getting user by id")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		if !roles.IsSuperAdmin && sub != id {
			if realmID, err := ihttp.GetRealmID(r); err != nil || realmID != user.RealmID {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		ihttp.Respond(w, user, http.StatusOK)
	}
}

func (s *Server) patchUserHandler() http.HandlerFunc {
	type patchUser struct {
		Username  *string      `json:"username" validate:"omitempty,gte=4,lte=32,alphanum"`
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
			ihttp.Error(w, http.StatusForbidden)
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
			targetUser, err := s.Store.GetUserByID(r.Context(), targetID)
			if err != nil {
				go s.Logger.WithError(err).Error("getting user")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}
			if realmID, err := ihttp.GetRealmID(r); err != nil || realmID != targetUser.RealmID {
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

		if ru.Username != nil {
			err := s.Store.CheckSimilarUsernameExists(r.Context(), *ru.Username, &targetID)
			if _, ok := errors.Cause(err).(store.ErrExists); ok {
				ihttp.Respond(w, err, http.StatusConflict)
				return
			} else if err != nil {
				go s.Logger.WithError(err).Error("checking whether similar user exists")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}
		}

		u := store.PatchUser{ID: targetID, Username: ru.Username, Roles: ru.Roles, FirstName: ru.FirstName, LastName: ru.LastName, Stars: ru.Stars}

		if ru.Password != nil {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*ru.Password), bcryptCost)
			if err != nil {
				go s.Logger.WithError(err).Error("hashing user password")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}

			hashedPasswordString := string(hashedPassword)
			u.HashedPassword = &hashedPasswordString
		}

		err = s.Store.PatchUser(r.Context(), u)

		if err != nil {
			switch errors.Cause(err).(type) {
			case store.ErrNoResults:
				ihttp.Error(w, http.StatusNotFound)
			case store.ErrFKeyViolation:
				ihttp.Error(w, http.StatusUnprocessableEntity)
			default:
				go s.Logger.WithError(err).Error("patching user")
				ihttp.Error(w, http.StatusInternalServerError)
			}

			return
		}

		w.WriteHeader(http.StatusNoContent)
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
			ihttp.Error(w, http.StatusForbidden)
			return
		}
		roles := ihttp.GetRoles(r)

		if id != requestID && !roles.IsAdmin && !roles.IsSuperAdmin {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		// Admins can only delete users in the same realm
		if id != requestID && !roles.IsSuperAdmin {
			targetUser, err := s.Store.GetUserByID(r.Context(), id)
			if err != nil {
				go s.Logger.WithError(err).Error("getting user")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}
			if realmID, err := ihttp.GetRealmID(r); err != nil || realmID != targetUser.RealmID {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		err = s.Store.DeleteUser(r.Context(), id)
		if err != nil {
			go s.Logger.WithError(err).Error("deleting user")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
