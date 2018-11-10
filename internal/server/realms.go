package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	validator "gopkg.in/go-playground/validator.v9"
)

type realm struct {
	Team       string `json:"team" validate:"required"`
	Name       string `json:"name" validate:"required"`
	PublicData bool   `json:"publicData"`
}

// createRealmHandler returns a handler to create a new realm.
func (s *Server) createRealmHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if roles := ihttp.GetRoles(r); !roles.IsSuperAdmin {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		var rr realm
		if err := json.NewDecoder(r.Body).Decode(&rr); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		if err := validator.New().Struct(rr); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		realm := store.Realm{Team: rr.Team, Name: rr.Name, PublicData: rr.PublicData}

		err := s.Store.InsertRealm(realm)
		if _, ok := err.(*store.ErrExists); ok {
			ihttp.Error(w, http.StatusConflict)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving realms")
			return
		}

		realmAdmin, adminPassword, err := s.createRealmAdmin(realm.Team)
		if _, ok := err.(*store.ErrExists); ok {
			ihttp.Respond(w, err, http.StatusConflict)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("creating realm admin")
			return
		}

		u := requestUser{
			baseUser: baseUser{
				Username: realmAdmin.Username,
				Password: adminPassword,
			},
			Realm:     realm.Team,
			FirstName: realmAdmin.FirstName,
			LastName:  realmAdmin.LastName,
			Roles:     realmAdmin.Roles,
		}

		ihttp.Respond(w, u, http.StatusOK)
	}
}

// realmsHandler returns a handler to get all realms.
func (s *Server) realmsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if roles := ihttp.GetRoles(r); !roles.IsSuperAdmin {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		realms, err := s.Store.GetRealms()
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving realms")
			return
		}

		ihttp.Respond(w, realms, http.StatusOK)
	}
}

// realmHandler returns a handler to get a specific realm.
func (s *Server) realmHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamKey := mux.Vars(r)["teamKey"]

		roles := ihttp.GetRoles(r)
		if !roles.IsSuperAdmin {
			if roles.IsAdmin {
				if ihttp.GetRealm(r) != teamKey {
					ihttp.Error(w, http.StatusForbidden)
					return
				}
			} else {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		realm, err := s.Store.GetRealm(teamKey)
		if _, ok := err.(*store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving realms")
			return
		}

		ihttp.Respond(w, realm, http.StatusOK)
	}
}

// patchRealmHandler returns a handler to modify a specific realm.
func (s *Server) patchRealmHandler() http.HandlerFunc {
	type patchRealm struct {
		Name       *string `json:"name" validate:"omitempty,gte=1,lte=32"`
		PublicData *bool   `json:"publicData"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		teamKey := mux.Vars(r)["teamKey"]

		roles := ihttp.GetRoles(r)
		if !roles.IsSuperAdmin {
			if roles.IsAdmin {
				if ihttp.GetRealm(r) != teamKey {
					ihttp.Error(w, http.StatusForbidden)
					return
				}
			} else {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		var pr patchRealm
		if err := json.NewDecoder(r.Body).Decode(&pr); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		if err := validator.New().Struct(pr); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		sr := store.PatchRealm{Team: teamKey, Name: pr.Name, PublicData: pr.PublicData}

		err := s.Store.PatchRealm(sr)
		if _, ok := err.(*store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			go s.Logger.WithError(err).Error("patching realm")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, nil, http.StatusNoContent)
	}
}

// deleteRealmHandler returns a handler to delete a specific realm.
func (s *Server) deleteRealmHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamKey := mux.Vars(r)["teamKey"]

		roles := ihttp.GetRoles(r)
		if !roles.IsSuperAdmin {
			if roles.IsAdmin {
				if ihttp.GetRealm(r) != teamKey {
					ihttp.Error(w, http.StatusForbidden)
					return
				}
			} else {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		err := s.Store.DeleteRealm(teamKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("deleting realms")
			return
		}

		ihttp.Respond(w, nil, http.StatusNoContent)
	}
}

func (s *Server) createRealmAdmin(teamKey string) (store.User, string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return store.User{}, "", errors.Wrap(err, "generating realm admin password")
	}
	adminPassword := base64.StdEncoding.EncodeToString(random)[:32]

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return store.User{}, "", errors.Wrap(err, "hashing realm admin password")
	}

	realmAdmin := store.User{
		Username:       "",
		HashedPassword: string(hashedPassword),
		Realm:          teamKey,
		FirstName:      "first",
		LastName:       "last",
		Roles:          store.Roles{IsAdmin: true, IsVerified: true, IsSuperAdmin: false},
	}

	alreadyExists := true
	for alreadyExists {
		if _, err := rand.Read(random); err != nil {
			return store.User{}, "", errors.Wrap(err, "generating realm admin username")
		}
		realmAdmin.Username = base64.StdEncoding.EncodeToString(random)[:32]
		err = s.Store.CreateUser(realmAdmin)
		_, alreadyExists = err.(*store.ErrExists)
	}
	return realmAdmin, adminPassword, err
}
