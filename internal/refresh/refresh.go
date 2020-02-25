// Package refresh is used for keeping the database up to date with TBA. The store methods here couold be
// updated to group resources into transactions, but it's performant enough as-is.
package refresh

import (
	"context"
	"errors"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/sirupsen/logrus"
)

// Service updates the store by polling TBA for the current year.
type Service struct {
	TBA    *tba.Service
	Store  *store.Service
	Logger *logrus.Logger
	Year   int
}

// Run starts the TBA updater service that will:
// * Update all events for the configured year, including matches, and rankings, every 15 minutes.
// * Update all teams every day.
// * Update all active event matches and rankings every 30 seconds.
func (s *Service) Run(ctx context.Context) {
	const (
		eventsInterval = time.Minute * 15
		activeInterval = time.Second * 30
		teamsInterval  = time.Hour * 24
	)

	events := make(chan []store.Event)
	storeEvents := make(chan []store.Event)
	matchEvents := make(chan string)
	rankingEvents := make(chan string)

	go func() {
		for eventGroup := range events {
			storeEvents <- eventGroup
			for _, event := range eventGroup {
				matchEvents <- event.Key
				rankingEvents <- event.Key
			}
		}

		close(storeEvents)
		close(matchEvents)
		close(rankingEvents)
	}()

	go s.fetchEvents(ctx, eventsInterval, events)
	go s.storeEvents(ctx, storeEvents)

	activeEvents := make(chan string)

	go func() {
		for event := range activeEvents {
			matchEvents <- event
			rankingEvents <- event
		}
	}()

	go s.seedActiveEvents(ctx, activeInterval, activeEvents)

	teams := make(chan []store.Team)
	go s.fetchTeams(ctx, teamsInterval, teams)
	go s.storeTeams(ctx, teams)

	matches := make(chan []store.Match)
	go s.fetchMatches(ctx, matchEvents, matches)
	go s.storeMatches(ctx, matches)

	rankings := make(chan []store.EventTeam)
	go s.fetchRankings(ctx, rankingEvents, rankings)
	s.storeRankings(ctx, rankings)
}

func (s *Service) fetchEvents(ctx context.Context, interval time.Duration, events chan<- []store.Event) {
	eventsTicker := time.NewTicker(interval)

	defer func() {
		eventsTicker.Stop()
		close(events)
	}()

	getEvents := func() {
		tbaEvents, err := s.TBA.GetEvents(ctx, s.Year)
		if err != nil && ctx.Err() != context.Canceled && !errors.Is(err, tba.ErrNotModified{}) {
			s.Logger.WithError(err).Errorf("unable get events from TBA for year %d", s.Year)
		}

		s.Logger.WithField("year", s.Year).WithField("count", len(tbaEvents)).Info("pulled events from TBA")

		events <- tbaEvents
	}

	getEvents()
	for {
		select {
		case <-eventsTicker.C:
			getEvents()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) seedActiveEvents(ctx context.Context, interval time.Duration, events chan<- string) {
	activeTicker := time.NewTicker(interval)

	defer func() {
		activeTicker.Stop()
		close(events)
	}()

	getEvents := func() {
		activeEvents, err := s.Store.GetActiveEvents(ctx)
		if err != nil && ctx.Err() != context.Canceled {
			s.Logger.WithError(err).Errorf("unable get active events %d", s.Year)
		}

		s.Logger.WithField("count", len(activeEvents)).Info("pulled active events")

		for _, event := range activeEvents {
			events <- event
		}
	}

	getEvents()
	for {
		select {
		case <-activeTicker.C:
			getEvents()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) storeEvents(ctx context.Context, events <-chan []store.Event) {
	for eventGroup := range events {
		err := s.Store.EventsUpsert(ctx, eventGroup)
		if err != nil && ctx.Err() != context.Canceled {
			s.Logger.WithError(err).Errorf("unable to upsert events")
		}

		s.Logger.WithField("count", len(eventGroup)).Info("stored events")
	}
}

func (s *Service) fetchTeams(ctx context.Context, interval time.Duration, teams chan<- []store.Team) {
	teamsTicker := time.NewTicker(interval)

	defer func() {
		teamsTicker.Stop()
		close(teams)
	}()

	getTeams := func() {
		tbaTeams, err := s.TBA.GetTeams(ctx)
		if err != nil && ctx.Err() != context.Canceled && !errors.Is(err, tba.ErrNotModified{}) {
			s.Logger.WithError(err).Errorf("unable get teams from TBA")
		}

		s.Logger.WithField("count", len(tbaTeams)).Info("pulled teams")

		teams <- tbaTeams
	}

	getTeams()
	for {
		select {
		case <-teamsTicker.C:
			getTeams()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) storeTeams(ctx context.Context, teams <-chan []store.Team) {
	for teamsGroup := range teams {
		err := s.Store.TeamsUpsert(ctx, teamsGroup)
		if err != nil && ctx.Err() != context.Canceled {
			s.Logger.WithError(err).Errorf("unable to upsert teams")
		}

		s.Logger.WithField("count", len(teamsGroup)).Info("stored teams")
	}
}

func (s *Service) fetchMatches(ctx context.Context, events <-chan string, matches chan<- []store.Match) {
	defer func() {
		close(matches)
	}()

	for eventKey := range events {
		tbaMatches, err := s.TBA.GetMatches(ctx, eventKey)
		if err != nil && ctx.Err() != context.Canceled && !errors.Is(err, tba.ErrNotModified{}) {
			s.Logger.WithError(err).Errorf("unable get matches from TBA for event %q", eventKey)
		}

		s.Logger.WithField("count", len(tbaMatches)).Info("pulled matches")

		matches <- tbaMatches
	}
}

func (s *Service) storeMatches(ctx context.Context, matches <-chan []store.Match) {
	for matchGroup := range matches {
		err := s.Store.UpdateTBAMatches(ctx, matchGroup)
		if err != nil && ctx.Err() != context.Canceled {
			s.Logger.WithError(err).Errorf("unable to upsert matches")
		}

		s.Logger.WithField("count", len(matchGroup)).Info("stored matches")
	}
}

func (s *Service) fetchRankings(ctx context.Context, eventKeys <-chan string, rankings chan<- []store.EventTeam) {
	defer func() {
		close(rankings)
	}()

	for eventKey := range eventKeys {
		tbaRankings, err := s.TBA.GetTeamRankings(ctx, eventKey)
		if err != nil && ctx.Err() != context.Canceled && !errors.Is(err, tba.ErrNotModified{}) {
			s.Logger.WithError(err).Errorf("unable get rankings from TBA for event %q", eventKey)
		}

		s.Logger.WithField("count", len(tbaRankings)).Info("pulled rankings")

		rankings <- tbaRankings
	}
}

func (s *Service) storeRankings(ctx context.Context, rankings <-chan []store.EventTeam) {
	for rankingGroup := range rankings {
		err := s.Store.EventTeamsUpsert(ctx, rankingGroup)
		if err != nil && ctx.Err() != context.Canceled {
			s.Logger.WithError(err).Errorf("unable to upsert rankings")
		}

		s.Logger.WithField("count", len(rankingGroup)).Info("stored rankings")
	}
}
