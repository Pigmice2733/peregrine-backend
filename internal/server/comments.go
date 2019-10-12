package server

import (
	"encoding/json"
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/gorilla/mux"
)

func (s *Server) getMatchTeamComments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		matchKey := vars["matchKey"]
		teamKey := vars["teamKey"]

		var comments []store.Comment
		var err error

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		comments, err = s.Store.GetMatchTeamCommentsForRealm(r.Context(), eventKey, matchKey, teamKey, realmID)

		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("getting comments")
			return
		}

		ihttp.Respond(w, comments, http.StatusOK)
	}
}

func (s *Server) getEventComments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		teamKey := vars["teamKey"]

		var comments []store.Comment
		var err error

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		comments, err = s.Store.GetEventTeamCommentsForRealm(r.Context(), eventKey, teamKey, realmID)

		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("getting comments")
			return
		}

		ihttp.Respond(w, comments, http.StatusOK)
	}
}

func (s *Server) putMatchTeamComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		matchKey := vars["matchKey"]
		teamKey := vars["teamKey"]

		var comment store.Comment
		if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		reporterID, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		created, err := s.Store.UpsertMatchTeamComment(r.Context(), eventKey, matchKey, teamKey, reporterID, comment.Comment)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("upserting comment")
			return
		}

		if created {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}

}
