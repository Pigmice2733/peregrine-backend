package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/jmoiron/sqlx"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

func (s *Server) reportsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventQuery := r.URL.Query().Get("event")
		matchQuery := r.URL.Query().Get("match")
		teamQuery := r.URL.Query().Get("team")
		reporterQuery := r.URL.Query().Get("reporter")

		var eventKey *string
		var matchKey *string
		var teamKey *string
		var reporterID *int64

		if eventQuery != "" {
			eventKey = &eventQuery
		}

		if matchQuery != "" {
			matchKey = &matchQuery
		}

		if teamQuery != "" {
			teamKey = &teamQuery
		}

		if id, err := strconv.ParseInt(reporterQuery, 10, 64); err == nil {
			reporterID = &id
		}

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		var reports []store.Report
		reports, err = s.Store.GetReports(r.Context(), eventKey, matchKey, teamKey, realmID, reporterID)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("getting reports")
			return
		}

		ihttp.Respond(w, reports, http.StatusOK)
	}
}

func (s *Server) reportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			ihttp.Error(w, http.StatusBadRequest)
			return
		}

		var realmID *int64
		userRealmID, err := ihttp.GetRealmID(r)
		if err == nil {
			realmID = &userRealmID
		}

		report, err := s.Store.GetReportForRealm(r.Context(), id, realmID)
		if errors.Is(err, store.ErrNoResults{}) {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("getting reports")
			return
		}

		ihttp.Respond(w, report, http.StatusOK)
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

		if report.ReporterID != nil {
			if reporterID != *report.ReporterID {
				ihttp.Error(w, http.StatusBadRequest)
				return
			}
		} else {
			report.ReporterID = &reporterID
		}

		var realmID int64
		realmID, err = ihttp.GetRealmID(r)
		if err != nil {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		if report.RealmID != nil {
			if realmID != *report.RealmID {
				ihttp.Error(w, http.StatusBadRequest)
				return
			}
		} else {
			report.RealmID = &realmID
		}

		var status int
		var reportID int64
		err = editReport(r.Context(), s.Store, nil, nil,
			func(tx *sqlx.Tx) error {
				// make sure team is present at match, and the event is visible to user
				present, err := s.Store.LockAlliance(r.Context(), tx, report.EventKey, report.MatchKey, report.TeamKey, &realmID)
				if err != nil {
					return err
				}

				if !present {
					return badRequestError{}
				}

				return nil
			},
			func(_ *store.Report, _ *store.User) error {
				return nil
			}, func(tx *sqlx.Tx) error {
				created, id, err := s.Store.UpsertReport(r.Context(), report)
				if created {
					status = http.StatusCreated
				} else {
					status = http.StatusOK
				}
				reportID = id
				return err
			})

		if errors.Is(err, badRequestError{}) {
			ihttp.Error(w, http.StatusBadRequest)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("upserting report")
			return
		}

		ihttp.Respond(w, reportID, status)
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

		report.ID = id

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

		err = editReport(r.Context(), s.Store, &id, report.ReporterID,
			func(tx *sqlx.Tx) error { return nil },
			func(oldReport *store.Report, targetUser *store.User) error {
				if oldReport == nil {
					return store.ErrNoResults{}
				}

				if !roles.IsSuperAdmin && !roles.IsAdmin {
					if report.ReporterID == nil || reporterID != *report.ReporterID ||
						oldReport.ReporterID == nil || reporterID != *oldReport.ReporterID {
						return forbiddenError{}
					}
				}

				if targetUser != nil {
					// report realm and reporter's realm must match if both specified
					if report.RealmID != nil && targetUser.RealmID != *report.RealmID {
						return badRequestError{}
					}

					if !roles.IsSuperAdmin && targetUser.RealmID != realmID {
						return forbiddenError{}
					}
				}

				if !roles.IsSuperAdmin {
					if (report.RealmID == nil || realmID != *report.RealmID) ||
						(oldReport.RealmID == nil || realmID != *oldReport.RealmID) {
						return forbiddenError{}
					}
				}

				return nil
			}, func(tx *sqlx.Tx) error {
				return s.Store.UpdateReportTx(r.Context(), tx, report)
			})

		if errors.Is(err, store.ErrNoResults{}) {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if errors.Is(err, store.ErrExists{}) {
			ihttp.Error(w, http.StatusBadRequest)
			return
		} else if errors.Is(err, forbiddenError{}) {
			ihttp.Error(w, http.StatusForbidden)
			return
		} else if errors.Is(err, badRequestError{}) {
			ihttp.Error(w, http.StatusBadRequest)
			return
		} else if err != nil {
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

		err = editReport(r.Context(), s.Store, &id, nil,
			func(tx *sqlx.Tx) error { return nil },
			func(report *store.Report, _ *store.User) error {
				if report == nil {
					return store.ErrNoResults{}
				}

				if roles.IsSuperAdmin {
					return nil
				}

				if report.RealmID != nil && userRealmID == *report.RealmID && roles.IsAdmin {
					return nil
				}

				if report.ReporterID != nil && userID == *report.ReporterID {
					return nil
				}

				return forbiddenError{}
			}, func(tx *sqlx.Tx) error {
				return s.Store.DeleteReportTx(r.Context(), tx, id)
			})

		if errors.Is(err, store.ErrNoResults{}) {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if errors.Is(err, forbiddenError{}) {
			ihttp.Error(w, http.StatusForbidden)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("unable to delete report")
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

		var filterYear *int
		if year, err := strconv.Atoi(r.URL.Query().Get("year")); err == nil {
			filterYear = &year
		}

		leaderboard, err := s.Store.GetLeaderboardForRealm(r.Context(), realmID, filterYear)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("getting leaderboard")
			return
		}

		ihttp.Respond(w, leaderboard, http.StatusOK)
	}
}

func editReport(ctx context.Context, s *store.Service, reportID, userID *int64,
	lockFunc func(tx *sqlx.Tx) error,
	validationFunc func(oldReport *store.Report, targetUser *store.User) error,
	editFunc func(tx *sqlx.Tx) error) error {
	return s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		var oldReport *store.Report
		if reportID != nil {
			report, err := s.LockReport(ctx, tx, *reportID)
			if err != nil {
				return err
			}

			oldReport = &report
		}

		var targetUser *store.User
		if userID != nil {
			user, err := s.LockUser(ctx, tx, *userID)
			if err != nil {
				return err
			}

			targetUser = &user
		}

		if err := lockFunc(tx); err != nil {
			return fmt.Errorf("unable to lock for editing report: %w", err)
		}

		if err := validationFunc(oldReport, targetUser); err != nil {
			return err
		}

		if err := editFunc(tx); err != nil {
			return fmt.Errorf("unable to edit report: %w", err)
		}

		return nil
	})
}
