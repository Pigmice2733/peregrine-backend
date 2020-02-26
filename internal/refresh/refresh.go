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

type eventMatches struct {
	EventKey string
	Matches  []store.Match
}

// Run starts the TBA updater service that will:
// * Update all events for the configured year, including matches, and rankings, every 15 minutes.
// * Update all teams every day.
// * Update all active event matches and rankings every 15 seconds.
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
	activeEvents := make(chan string)

	go func() {
		defer func() {
			close(storeEvents)
			close(matchEvents)
			close(rankingEvents)
		}()

		for {
			select {
			case event := <-activeEvents:
				matchEvents <- event
				rankingEvents <- event
			case eventGroup := <-events:
				storeEvents <- eventGroup
				for _, event := range eventGroup {
					matchEvents <- event.Key
					rankingEvents <- event.Key
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go s.fetchEvents(ctx, eventsInterval, events)
	go s.storeEvents(ctx, storeEvents)
	go s.seedActiveEvents(ctx, activeInterval, activeEvents)

	teams := make(chan []store.Team)
	go s.fetchTeams(ctx, teamsInterval, teams)
	go s.storeTeams(ctx, teams)

	matches := make(chan eventMatches)
	go s.fetchMatches(ctx, matchEvents, matches)
	go s.storeMatches(ctx, matches)

	rankings := make(chan []store.EventTeam)
	go s.fetchRankings(ctx, rankingEvents, rankings)
	s.storeRankings(ctx, rankings)
}

func (s *Service) fetchEvents(ctx context.Context, interval time.Duration, events chan<- []store.Event) {
	const timeout = time.Second * 20

	eventsTicker := time.NewTicker(interval)

	defer func() {
		eventsTicker.Stop()
		close(events)
	}()

	getEvents := func() {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		tbaEvents, err := s.TBA.GetEvents(timeoutContext, s.Year)
		if errors.Is(err, tba.ErrNotModified{}) {
			return
		} else if err != nil {
			s.Logger.WithError(err).Errorf("unable get events from TBA for year %d", s.Year)
			return
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
	const timeout = time.Second * 10

	activeTicker := time.NewTicker(interval)

	defer func() {
		activeTicker.Stop()
		close(events)
	}()

	getEvents := func() {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		activeEvents, err := s.Store.GetActiveEvents(timeoutContext)
		if err != nil {
			s.Logger.WithError(err).Errorf("unable get active events %d", s.Year)
			return
		}

		s.Logger.WithField("count", len(activeEvents)).Info("pulled active events")
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
	const timeout = time.Second * 10

	for eventGroup := range events {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		err := s.Store.EventsUpsert(timeoutContext, eventGroup)
		if err != nil {
			s.Logger.WithError(err).Errorf("unable to upsert events")
			return
		}

		s.Logger.WithField("count", len(eventGroup)).Info("stored events")
	}
}

func (s *Service) fetchTeams(ctx context.Context, interval time.Duration, teams chan<- []store.Team) {
	const timeout = time.Second * 20

	teamsTicker := time.NewTicker(interval)

	defer func() {
		teamsTicker.Stop()
		close(teams)
	}()

	getTeams := func() {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		tbaTeams, err := s.TBA.GetTeams(timeoutContext)
		if errors.Is(err, tba.ErrNotModified{}) {
			return
		} else if err != nil {
			s.Logger.WithError(err).Errorf("unable get teams from TBA")
			return
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
	const timeout = time.Second * 10

	for teamsGroup := range teams {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		err := s.Store.TeamsUpsert(timeoutContext, teamsGroup)
		if err != nil {
			s.Logger.WithError(err).Errorf("unable to upsert teams")
			continue
		}

		s.Logger.WithField("count", len(teamsGroup)).Info("stored teams")
	}
}

func (s *Service) fetchMatches(ctx context.Context, events <-chan string, matches chan<- eventMatches) {
	const timeout = time.Second * 10

	defer func() {
		close(matches)
	}()

	for eventKey := range events {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		tbaMatches, err := s.TBA.GetMatches(timeoutContext, eventKey)
		if errors.Is(err, tba.ErrNotModified{}) {
			continue
		} else if err != nil {
			s.Logger.WithError(err).Errorf("unable get matches from TBA for event %q", eventKey)
			continue
		}

		s.Logger.WithField("count", len(tbaMatches)).Info("pulled matches")

		matches <- eventMatches{
			EventKey: eventKey,
			Matches:  tbaMatches,
		}
	}
}

func (s *Service) storeMatches(ctx context.Context, eventMatches <-chan eventMatches) {
	const timeout = time.Second * 10

	for matches := range eventMatches {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		err := s.Store.UpdateTBAMatches(timeoutContext, matches.Matches)
		if err != nil {
			s.Logger.WithError(err).Errorf("unable to upsert matches")
			continue
		}

		err = s.Store.MarkMatchesDeleted(ctx, matches.EventKey, matches.Matches)
		if err != nil {
			s.Logger.WithError(err).Errorf("unable to mark matches deleted matches")
		}

		s.Logger.WithField("count", len(matches.Matches)).Info("stored matches")
	}
}

func (s *Service) fetchRankings(ctx context.Context, eventKeys <-chan string, rankings chan<- []store.EventTeam) {
	const timeout = time.Second * 10

	defer func() {
		close(rankings)
	}()

	for eventKey := range eventKeys {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		tbaRankings, err := s.TBA.GetTeamRankings(timeoutContext, eventKey)
		if errors.Is(err, tba.ErrNotModified{}) {
			continue
		} else if err != nil {
			s.Logger.WithError(err).Errorf("unable get rankings from TBA for event %q", eventKey)
			continue
		}

		s.Logger.WithField("count", len(tbaRankings)).Info("pulled rankings")
		rankings <- tbaRankings
	}
}

func (s *Service) storeRankings(ctx context.Context, rankings <-chan []store.EventTeam) {
	const timeout = time.Second * 10

	for rankingGroup := range rankings {
		timeoutContext, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		err := s.Store.EventTeamsUpsert(timeoutContext, rankingGroup)
		if err != nil {
			s.Logger.WithError(err).Errorf("unable to upsert rankings")
			continue
		}

		s.Logger.WithField("count", len(rankingGroup)).Info("stored rankings")
	}
}
