package server

import (
	"context"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/pkg/errors"
)

// BeginTbaBackgroundUpdates begins periodically updating all TBA data in background
func (s *Server) BeginTbaBackgroundUpdates() {
	ctx := context.Background()

	inactiveUpdates := time.Tick(15 * time.Minute)
	activeUpdates := time.Tick(1 * time.Minute)

	s.Logger.Info("Seeding TBA data")
	go s.updateEvents(ctx)
	go s.updateTeams(ctx)
	go s.updatePerEventData(ctx, false)

	for {
		select {
		case _ = <-inactiveUpdates:
			go s.updateEvents(ctx)
			go s.updateTeams(ctx)
			go s.updatePerEventData(ctx, false)
		case _ = <-activeUpdates:
			go s.updatePerEventData(ctx, true)
		}
	}
}

func (s *Server) updatePerEventData(ctx context.Context, activeOnly bool) {
	var events []store.Event
	var err error

	if activeOnly {
		events, err = s.Store.GetActiveEvents(ctx, false)
	} else {
		events, err = s.Store.GetEvents(ctx, false)
	}

	if err != nil {
		s.Logger.WithError(err).Error("getting events from store")
	}

	go s.updateMatches(ctx, events)
	go s.updateEventTeamRankings(ctx, events)
}

// Get new event data from TBA, upsert event data into database.
func (s *Server) updateEvents(ctx context.Context) {
	events, err := s.TBA.GetEvents(ctx, s.Year)
	if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
		return
	} else if err != nil {
		s.Logger.WithError(err).Errorf("unable get events from TBA for year %d", s.Year)
		return
	}

	if err := s.Store.EventsUpsert(ctx, events); err != nil {
		s.Logger.WithError(err).Errorf("upserting events")
		return
	}

	if err := s.Store.MarkEventsDeleted(ctx, events); err != nil {
		s.Logger.WithError(err).Errorf("marking missing events deleted")
	}
}

// Get new teams data from TBA, upsert teams data into database.
func (s *Server) updateTeams(ctx context.Context) {
	teams, err := s.TBA.GetTeams(ctx)
	if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
		return
	} else if err != nil {
		s.Logger.WithError(err).Errorf("retrieving teams")
		return
	}

	if err := s.Store.TeamsUpsert(ctx, teams); err != nil {
		s.Logger.WithError(err).Errorf("upserting teams")
	}
}

// Get new match data from TBA for a particular event, upsert match data into database.
func (s *Server) updateMatches(ctx context.Context, events []store.Event) {
	for _, event := range events {
		var fullMatches []store.Match
		var err error

		if !event.TBADeleted {
			fullMatches, err = s.TBA.GetMatches(ctx, event.Key)
			if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
				continue
			} else if err != nil {
				s.Logger.WithError(err).Errorf("unable to fetch matches from TBA")
				return
			}

			if err := s.Store.UpdateTBAMatches(ctx, event.Key, fullMatches); err != nil {
				s.Logger.WithError(err).Errorf("unable to update matches")
				return
			}
		}

		err = s.Store.MarkMatchesDeleted(ctx, event.Key, fullMatches)
		if err != nil {
			s.Logger.WithError(err).Errorf("unable to mark deleted matches")
			return
		}
	}
}

// Get new team rankings data from TBA for a particular event, upsert data into database.
func (s *Server) updateEventTeamRankings(ctx context.Context, events []store.Event) {
	for _, event := range events {
		teams, err := s.TBA.GetTeamRankings(ctx, event.Key)
		if _, ok := errors.Cause(err).(tba.ErrNotModified); ok {
			continue
		} else if err != nil {
			s.Logger.WithError(err).Error("getting team ranking data from TBA")
			return
		}

		if err := s.Store.EventTeamsUpsert(ctx, teams); err != nil {
			s.Logger.WithError(err).Error("upserting team ranking data")
			return
		}
	}
}
