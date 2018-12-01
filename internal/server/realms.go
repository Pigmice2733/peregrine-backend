package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	validator "gopkg.in/go-playground/validator.v9"
)

type realm struct {
	Name         string `json:"name" validate:"required"`
	ShareReports bool   `json:"shareReports"`
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

		realm := store.Realm{Name: rr.Name, ShareReports: rr.ShareReports}

		id, err := s.Store.InsertRealm(realm)
		if _, ok := errors.Cause(err).(store.ErrExists); ok {
			ihttp.Error(w, http.StatusConflict)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving realms")
			return
		}

		ihttp.Respond(w, id, http.StatusOK)
	}
}

// realmsHandler returns a handler to get all realms.
func (s *Server) realmsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roles := ihttp.GetRoles(r)

		var realms []store.Realm
		var err error

		if roles.IsSuperAdmin {
			realms, err = s.Store.GetRealms()
			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				go s.Logger.WithError(err).Error("retrieving realms")
				return
			}
		} else {
			realms, err = s.Store.GetPublicRealms()
			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				go s.Logger.WithError(err).Error("retrieving realms")
				return
			}
			var userRealm int64
			userRealm, err = ihttp.GetRealmID(r)
			if err != nil {
				ihttp.Respond(w, realms, http.StatusOK)
				return
			}
			var realm store.Realm
			realm, err = s.Store.GetRealm(userRealm)
			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				go s.Logger.WithError(err).Error(fmt.Sprintf("retrieving realm %d", userRealm))
				return
			}
			if !realm.ShareReports {
				realms = append(realms, realm)
			}
		}

		ihttp.Respond(w, realms, http.StatusOK)
	}
}

// realmHandler returns a handler to get a specific realm.
func (s *Server) realmHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		realm, err := s.Store.GetRealm(id)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("retrieving realms")
			return
		}

		roles := ihttp.GetRoles(r)
		if !roles.IsSuperAdmin && !realm.ShareReports {
			userRealm, err := ihttp.GetRealmID(r)
			if err != nil {
				ihttp.Error(w, http.StatusUnauthorized)
				return
			}
			if userRealm != id {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		ihttp.Respond(w, realm, http.StatusOK)
	}
}

// patchRealmHandler returns a handler to modify a specific realm.
func (s *Server) patchRealmHandler() http.HandlerFunc {
	type patchRealm struct {
		Name         *string `json:"name" validate:"omitempty,gte=1,lte=32"`
		ShareReports *bool   `json:"shareReports"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		roles := ihttp.GetRoles(r)
		if !roles.IsSuperAdmin {
			if roles.IsAdmin {
				userRealm, err := ihttp.GetRealmID(r)
				if err != nil {
					ihttp.Error(w, http.StatusUnauthorized)
					return
				}
				if userRealm != id {
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

		sr := store.PatchRealm{ID: id, Name: pr.Name, ShareReports: pr.ShareReports}

		err = s.Store.PatchRealm(sr)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
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
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		roles := ihttp.GetRoles(r)
		if !roles.IsSuperAdmin {
			if roles.IsAdmin {
				userRealm, err := ihttp.GetRealmID(r)
				if err != nil {
					ihttp.Error(w, http.StatusUnauthorized)
					return
				}
				if userRealm != id {
					ihttp.Error(w, http.StatusForbidden)
					return
				}
			} else {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		err = s.Store.DeleteRealm(id)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("deleting realms")
			return
		}

		ihttp.Respond(w, nil, http.StatusNoContent)
	}
}
