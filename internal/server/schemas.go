package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
)

// StatDescription escribes a single statistic in a schema
type StatDescription struct {
	Name string     `json:"name"`
	ID   *uuid.UUID `json:"id"`
	Type string     `json:"type"`
}

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

		if err := populateStatIDs(&schema.Auto); err != nil {
			go s.Logger.WithError(err).Error("populating auto stat IDs")
			ihttp.Error(w, http.StatusInternalServerError)
			return
		}
		if err := populateStatIDs(&schema.Teleop); err != nil {
			go s.Logger.WithError(err).Error("populating teleop stat IDs")
			ihttp.Error(w, http.StatusInternalServerError)
			return
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

func populateStatIDs(j *json.RawMessage) error {
	if j == nil {
		return fmt.Errorf("can't populate stat IDs on nil JSON raw message")
	}

	stats, err := jsonToStatDescription(*j)
	if err != nil {
		return err
	}

	var populatedStats []StatDescription

	for _, stat := range stats {
		if stat.ID == nil {
			id, err := uuid.NewV4()
			if err != nil {
				return err
			}
			stat.ID = &id
			populatedStats = append(populatedStats, stat)
		}
	}

	*j, err = statDescriptionToJSON(populatedStats)
	return err
}

func jsonToStatDescription(j json.RawMessage) ([]StatDescription, error) {
	var stats []StatDescription
	return stats, json.Unmarshal(j, &stats)
}

func statDescriptionToJSON(s []StatDescription) (json.RawMessage, error) {
	return json.Marshal(s)
}
