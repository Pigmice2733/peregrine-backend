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

// createRealmHandler returns a handler to create a new realm.
func (s *Server) createRealmHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if roles := ihttp.GetRoles(r); !roles.IsSuperAdmin {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		var realm store.Realm
		if err := json.NewDecoder(r.Body).Decode(&realm); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		if err := validator.New().Struct(realm); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		id, err := s.Store.InsertRealm(r.Context(), realm)
		if _, ok := errors.Cause(err).(store.ErrExists); ok {
			ihttp.Error(w, http.StatusConflict)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("inserting realms")
			return
		}

		ihttp.Respond(w, id, http.StatusCreated)
	}
}

// realmsHandler returns a handler to get all realms.
func (s *Server) realmsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roles := ihttp.GetRoles(r)

		var realms []store.Realm
		var err error

		if roles.IsSuperAdmin {
			realms, err = s.Store.GetRealms(r.Context())
			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				go s.Logger.WithError(err).Error("retrieving realms")
				return
			}
		} else {
			realms, err = s.Store.GetPublicRealms(r.Context())
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
			realm, err = s.Store.GetRealm(r.Context(), userRealm)
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

		realm, err := s.Store.GetRealm(r.Context(), id)
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
				ihttp.Error(w, http.StatusForbidden)
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

// updateRealmHandler returns a handler to update a specific realm.
func (s *Server) updateRealmHandler() http.HandlerFunc {
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
					ihttp.Error(w, http.StatusForbidden)
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

		var realm store.Realm
		if err := json.NewDecoder(r.Body).Decode(&realm); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		if err := validator.New().Struct(realm); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		err = s.Store.UpdateRealm(r.Context(), realm)
		if _, ok := errors.Cause(err).(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("updating realms")
			return
		}

		ihttp.Respond(w, id, http.StatusNoContent)
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
					ihttp.Error(w, http.StatusForbidden)
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

		err = s.Store.DeleteRealm(r.Context(), id)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("deleting realms")
			return
		}

		ihttp.Respond(w, nil, http.StatusNoContent)
	}
}
