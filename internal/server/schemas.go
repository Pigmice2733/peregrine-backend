package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
				go s.Logger.WithError(err).Error("getting realmID")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}
			schema.RealmID = &realmID
		} else {
			schema.RealmID = nil
		}

		err := s.Store.CreateSchema(schema)
		if _, ok := err.(*store.ErrExists); ok {
			ihttp.Respond(w, err, http.StatusConflict)
			return
		} else if err != nil {
			go s.Logger.WithError(err).Error("creating schema")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		// If a new schema is created for a specific year, invalidate the events
		// route so events (and their schemas) will be updated.
		if schema.Year != nil {
			expiredUpdate := time.Now().Add(-(expiry + 1) * time.Hour)
			s.eventsLastUpdate = &expiredUpdate
		}

		ihttp.Respond(w, nil, http.StatusCreated)
	}
}

func (s *Server) getSchemasHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var schemas []store.Schema
		var err error
		roles := ihttp.GetRoles(r)
		if roles.IsSuperAdmin {
			schemas, err = s.Store.GetSchemas()
			if err != nil {
				go s.Logger.WithError(err).Error("getting schemas")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}
		} else {
			var realmID *int64
			realm, err := ihttp.GetRealmID(r)
			if err != nil {
				realmID = nil
			} else {
				realmID = &realm
			}
			schemas, err = s.Store.GetVisibleSchemas(realmID)
			if err != nil {
				go s.Logger.WithError(err).Error("getting schemas")
				ihttp.Error(w, http.StatusInternalServerError)
				return
			}
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

		schema, err := s.Store.GetSchemaByID(id)
		if _, ok := err.(*store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			go s.Logger.WithError(err).Error("getting schema by id")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		roles := ihttp.GetRoles(r)
		var realmID *int64
		realm, err := ihttp.GetRealmID(r)
		if err != nil {
			realmID = nil
		} else {
			realmID = &realm
		}

		if schema.Year == nil && !roles.IsSuperAdmin && schema.RealmID != realmID {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		ihttp.Respond(w, schema, http.StatusOK)
	}
}

func (s *Server) getSchemaByYearHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		year, err := strconv.ParseInt(mux.Vars(r)["year"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		schema, err := s.Store.GetSchemaByYear(int(year))
		if _, ok := err.(*store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			go s.Logger.WithError(err).Error("getting schema by year")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}

		ihttp.Respond(w, schema, http.StatusOK)
	}
}
