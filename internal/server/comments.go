package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
)

func (s *Server) getComments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("bla")
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		partialMatchKey := vars["matchKey"]
		teamKey := vars["teamKey"]

		if _, err := s.Store.CheckTBAEventKeyExists(r.Context(), eventKey); err != nil {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey := fmt.Sprintf("%s_%s", eventKey, partialMatchKey)

		exists, err := s.Store.CheckMatchKeyExists(matchKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("checking that match exists")
			return
		}
		if !exists {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		comments, err := s.Store.GetComments(r.Context(), matchKey, teamKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("getting comments")
			return
		}

		ihttp.Respond(w, comments, http.StatusOK)
	}
}

func (s *Server) putComments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		partialMatchKey := vars["matchKey"]
		teamKey := vars["teamKey"]

		if _, err := s.Store.CheckTBAEventKeyExists(r.Context(), eventKey); err != nil {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey := fmt.Sprintf("%s_%s", eventKey, partialMatchKey)

		exists, err := s.Store.CheckMatchKeyExists(matchKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("checking that match exists")
			return
		} else if !exists {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		var comment store.Comment
		if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		comment.MatchKey = matchKey
		comment.TeamKey = teamKey

		reporterID, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}
		comment.ReporterID = &reporterID

		var realmID int64
		realmID, err = ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}
		comment.RealmID = &realmID

		created, err := s.Store.UpsertComment(r.Context(), comment)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("upserting comment")
			return
		}

		if created {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}

}
