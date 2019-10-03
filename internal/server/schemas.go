package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
)

func (s *Server) createSchemaHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var schema store.Schema
		if err := json.NewDecoder(r.Body).Decode(&schema); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		roles := ihttp.GetRoles(r)
		if schema.Year != nil && !roles.IsSuperAdmin {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		if schema.Year == nil {
			realmID, err := ihttp.GetRealmID(r)
			if err != nil {
				s.Logger.WithError(err).Error("getting realmID")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}

			schema.RealmID = &realmID
		} else {
			schema.RealmID = nil
		}

		err := s.Store.CreateSchema(r.Context(), schema)
		if _, ok := err.(store.ErrExists); ok {
			ihttp.Respond(w, err, http.StatusConflict)
			return
		} else if err != nil {
			s.Logger.WithError(err).Error("creating schema")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func (s *Server) getSchemasHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if yearQuery := r.URL.Query().Get("year"); yearQuery != "" {
			year, err := strconv.Atoi(yearQuery)
			if err != nil {
				ihttp.Error(w, http.StatusBadRequest)
				return
			}

			schema, err := s.Store.GetSchemaByYear(r.Context(), year)
			if _, ok := err.(store.ErrNoResults); ok {
				ihttp.Error(w, http.StatusNotFound)
				return
			} else if err != nil {
				s.Logger.WithError(err).Error("getting schema by year")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}

			ihttp.Respond(w, []store.Schema{schema}, http.StatusOK)
			return
		}

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		schemas, err := s.Store.GetSchemasForRealm(r.Context(), realmID)
		if err != nil {
			s.Logger.WithError(err).Error("getting schemas")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, schemas, http.StatusOK)
	}
}

func (s *Server) getSchemaByIDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		schema, err := s.Store.GetSchemaByID(r.Context(), id)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			s.Logger.WithError(err).Error("getting schema by id")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		roles := ihttp.GetRoles(r)

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		if schema.Year == nil && !roles.IsSuperAdmin && schema.RealmID != realmID {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		ihttp.Respond(w, schema, http.StatusOK)
	}
}
