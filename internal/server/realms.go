package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"errors"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	validator "gopkg.in/go-playground/validator.v9"
)

// createRealmHandler returns a handler to create a new realm.
func (s *Server) createRealmHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if errors.Is(err, store.ErrExists{}) {
			ihttp.Error(w, http.StatusConflict)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("inserting realms")
			return
		}

		realm.ID = id

		ihttp.Respond(w, realm, http.StatusCreated)
	}
}

// realmsHandler returns a handler to get all realms.
func (s *Server) realmsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realms, err := s.Store.GetRealms(r.Context())
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving realms")
			return
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
		if errors.Is(err, store.ErrNoResults{}) {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving realms")
			return
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

		var realm store.Realm
		if err := json.NewDecoder(r.Body).Decode(&realm); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		if err := validator.New().Struct(realm); err != nil {
			ihttp.Respond(w, err, http.StatusUnprocessableEntity)
			return
		}

		roles := ihttp.GetRoles(r)
		userRealmID, err := ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		existed, err := editRealm(r.Context(), s.Store, roles, userRealmID, id, func(tx *sqlx.Tx) error {
			if err := s.Store.UpdateRealmTx(r.Context(), tx, realm); err != nil {
				return fmt.Errorf("unable to update realm %d: %w", realm.ID, err)
			}

			return nil
		})
		if errors.Is(err, forbiddenError{}) {
			ihttp.Error(w, http.StatusForbidden)
			return
		} else if err != nil {
			s.Logger.WithError(err).Error("unable to upsert realm")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		if existed {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
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
		userRealmID, err := ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		existed, err := editRealm(r.Context(), s.Store, roles, userRealmID, id, func(tx *sqlx.Tx) error {
			if err := s.Store.DeleteRealmTx(r.Context(), tx, id); err != nil {
				return fmt.Errorf("unable to delete realm %d: %w", id, err)
			}

			return nil
		})
		if errors.Is(err, forbiddenError{}) {
			ihttp.Error(w, http.StatusForbidden)
			return
		} else if err != nil {
			s.Logger.WithError(err).Error("unable to delete match")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		if existed {
			w.WriteHeader(http.StatusNoContent)
		} else {
			ihttp.Error(w, http.StatusNotFound)
		}
	}
}

func editRealm(ctx context.Context, sto *store.Service, roles store.Roles, userRealmID, realmID int64, editFunc func(tx *sqlx.Tx) error) (existed bool, err error) {
	existed = true

	err = sto.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		if err := sto.ExclusiveLockRealmsTx(ctx, tx); err != nil {
			return fmt.Errorf("unable to lock realms for edit: %w", err)
		}

		exists, err := sto.GetRealmExistsTx(ctx, tx, realmID)
		if err != nil {
			return fmt.Errorf("unable to get whether realm exists: %w", err)
		}

		existed = exists

		if !roles.IsAdmin && !roles.IsSuperAdmin {
			return forbiddenError{errors.New("only super-admins and admins can edit realms")}
		}

		if !roles.IsSuperAdmin && userRealmID != realmID {
			return forbiddenError{errors.New("realm admins can only edit realms they belong to")}
		}

		if err := editFunc(tx); err != nil {
			return fmt.Errorf("unable to edit realm: %w", err)
		}

		return nil
	})

	return existed, err
}
