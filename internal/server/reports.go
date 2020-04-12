package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/Pigmice2733/peregrine-backend/internal/store"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

func (s *Server) getReportsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventQuery := r.URL.Query().Get("event")
		matchQuery := r.URL.Query().Get("match")
		teamQuery := r.URL.Query().Get("team")

		eventSpecified := eventQuery != ""
		matchSpecified := matchQuery != ""
		teamSpecified := teamQuery != ""

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		var reports []store.Report
		if eventSpecified && matchSpecified && teamSpecified {
			reports, err = s.Store.GetMatchTeamReportsForRealm(r.Context(), eventQuery, matchQuery, teamQuery, realmID)
			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				s.Logger.WithError(err).Error("getting reports")
				return
			}
		} else if eventSpecified && teamSpecified {
			reports, err = s.Store.GetEventTeamReportsForRealm(r.Context(), eventQuery, teamQuery, realmID)
			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				s.Logger.WithError(err).Error("getting reports")
				return
			}
		} else {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		ihttp.Respond(w, reports, http.StatusOK)
	}
}

func (s *Server) postReportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var report store.Report
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		reporterID, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}
		report.ReporterID = &reporterID

		var realmID int64
		realmID, err = ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}
		report.RealmID = &realmID

		// make sure team is present at match, and the event is visible to user
		present, err := s.Store.IsTeamPresentAtMatch(r.Context(), report.EventKey, report.MatchKey, report.TeamKey, &realmID)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("checking team at match")
			return
		}

		if !present {
			ihttp.Error(w, http.StatusNotFound)
			return
		}

		created, id, err := s.Store.UpsertReport(r.Context(), report)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("upserting report")
			return
		}

		var status int
		if created {
			status = http.StatusCreated
		} else {
			status = http.StatusOK
		}

		ihttp.Respond(w, id, status)
	}
}

func (s *Server) putReportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		var report store.Report
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			ihttp.Error(w, http.StatusUnprocessableEntity)
			return
		}

		roles := ihttp.GetRoles(r)

		reporterID, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		var realmID int64
		realmID, err = ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		oldReport, err := s.Store.GetReportByID(r.Context(), id)
		if errors.Is(err, store.ErrNoResults{}) {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("unable to retrieve report")
			return
		}

		report.ID = id

		if !roles.IsSuperAdmin && !roles.IsAdmin {
			if report.ReporterID == nil || reporterID != *report.ReporterID ||
				oldReport.ReporterID == nil || reporterID != *oldReport.ReporterID {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		} else {
			if report.ReporterID != nil {
				targetUser, err := s.Store.GetUserByID(r.Context(), *report.ReporterID)
				if errors.Is(err, store.ErrNoResults{}) {
					ihttp.Error(w, http.StatusNotFound)
					return
				} else if err != nil {
					ihttp.Error(w, http.StatusInternalServerError)
					s.Logger.WithError(err).Error("unable to retrieve user")
					return
				}

				// report realm and reporter's realm must match if both specified
				if report.RealmID != nil && targetUser.RealmID != *report.RealmID {
					ihttp.Error(w, http.StatusBadRequest)
					return
				}

				if !roles.IsSuperAdmin && targetUser.RealmID != realmID {
					ihttp.Error(w, http.StatusForbidden)
					return
				}
			}
		}

		if !roles.IsSuperAdmin {
			if (report.RealmID != nil && realmID != *report.RealmID) ||
				(oldReport.RealmID != nil && realmID != *oldReport.RealmID) {
				ihttp.Error(w, http.StatusForbidden)
				return
			}
		}

		err = s.Store.UpdateReport(r.Context(), report)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("updating report")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// deleteReportHandler returns a handler to delete a specific report.
func (s *Server) deleteReportHandler() http.HandlerFunc {
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

		userID, err := ihttp.GetSubject(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		report, err := s.Store.GetReportByID(r.Context(), id)
		if errors.Is(err, store.ErrNoResults{}) {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("unable to retrieve report")
			return
		}

		if !(roles.IsSuperAdmin) &&
			!(roles.IsAdmin && report.RealmID != nil && userRealmID == *report.RealmID) &&
			!(report.ReporterID != nil && userID == *report.ReporterID) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		err = s.Store.DeleteReportByID(r.Context(), id)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("unable to retrieve report")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) leaderboardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realmID, err := ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		leaderboard, err := s.Store.GetLeaderboardForRealm(r.Context(), realmID)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("getting leaderboard")
			return
		}

		ihttp.Respond(w, leaderboard, http.StatusOK)
	}
}
