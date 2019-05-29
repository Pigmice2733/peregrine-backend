package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/fharding1/gemux"
)

// this is a hack because match keys are stored weirdly right now
func trimMatchKey(key string) string {
	parts := strings.Split(key, "_")
	if len(parts) != 2 {
		return ""
	}

	return parts[1]
}

func (s *Server) getMatchTeamComments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := gemux.PathParameter(r.Context(), 0)
		partialMatchKey := gemux.PathParameter(r.Context(), 1)
		teamKey := gemux.PathParameter(r.Context(), 2)

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

		comments, err := s.Store.GetMatchTeamComments(r.Context(), matchKey, teamKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("getting comments")
			return
		}

		// this is a hack since match keys are stored weirdly right now
		for i, c := range comments {
			comments[i].MatchKey = trimMatchKey(c.MatchKey)
		}

		ihttp.Respond(w, comments, http.StatusOK)
	}
}

func (s *Server) getEventComments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := gemux.PathParameter(r.Context(), 0)
		teamKey := gemux.PathParameter(r.Context(), 1)

		if _, err := s.Store.CheckTBAEventKeyExists(r.Context(), eventKey); err != nil {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		comments, err := s.Store.GetEventComments(r.Context(), eventKey, teamKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			go s.Logger.WithError(err).Error("getting comments")
			return
		}

		// this is a hack since match keys are stored weirdly right now
		for i, c := range comments {
			comments[i].MatchKey = trimMatchKey(c.MatchKey)
		}

		ihttp.Respond(w, comments, http.StatusOK)
	}
}

func (s *Server) putMatchTeamComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventKey := gemux.PathParameter(r.Context(), 0)
		partialMatchKey := gemux.PathParameter(r.Context(), 1)
		teamKey := gemux.PathParameter(r.Context(), 2)

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

		comment.EventKey = eventKey
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

		created, err := s.Store.UpsertMatchTeamComment(r.Context(), comment)
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
