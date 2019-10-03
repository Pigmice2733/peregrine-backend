package tbaupdater

import (
	"context"
	"errors"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/sirupsen/logrus"
)

// Service provides methods for running periodic background updates of TBA data
type Service struct {
	TBA    *tba.Service
	Store  *store.Service
	Logger *logrus.Logger
	Year   int
	cancel *context.CancelFunc
}

// Begin starts periodic updates of TBA data
func (s *Service) Begin() {
	if s.cancel == nil {
		s.Logger.Info("beginning background TBA updates")
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = &cancel
		go s.run(ctx)
	}
}

// End stops periodic updates of TBA data, if running
func (s *Service) End() {
	if s.cancel != nil {
		s.Logger.Info("ending background TBA updates")
		(*s.cancel)()
		s.cancel = nil
	}
}

// run periodically updates all TBA data in background
func (s *Service) run(ctx context.Context) {
	inactiveUpdates := time.NewTicker(15 * time.Minute)
	activeUpdates := time.NewTicker(1 * time.Minute)
	defer inactiveUpdates.Stop()
	defer activeUpdates.Stop()

	s.Logger.Info("seeding TBA data")
	go s.updateEvents(ctx)
	go s.updateTeams(ctx)
	go s.updatePerEventData(ctx, false)

	for {
		select {
		case <-ctx.Done():
			return
		case <-inactiveUpdates.C:
			go s.updateEvents(ctx)
			go s.updateTeams(ctx)
			go s.updatePerEventData(ctx, false)
		case <-activeUpdates.C:
			go s.updatePerEventData(ctx, true)
		}
	}
}

// updatePerEventData updates all data that is tied to individual events, such as match and team ranking data
// activeOnly specifies whether only data for active (currently happening) events should be updated
func (s *Service) updatePerEventData(ctx context.Context, activeOnly bool) {
	var events []store.Event
	var err error

	if activeOnly {
		events, err = s.Store.GetActiveEvents(ctx, false)
	} else {
		events, err = s.Store.GetEvents(ctx, false)
	}

	if err != nil {
		if ctx.Err() != context.Canceled {
			s.Logger.WithError(err).Error("getting events from store")
		}
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	go s.updateMatches(ctx, events)
	go s.updateEventTeamRankings(ctx, events)
}

// updateEvents gets new event data from TBA and upserts that event data into the database.
func (s *Service) updateEvents(ctx context.Context) {
	events, err := s.TBA.GetEvents(ctx, s.Year)
	if errors.Is(err, tba.ErrNotModified{}) {
		return
	} else if err != nil {
		if ctx.Err() != context.Canceled {
			s.Logger.WithError(err).Errorf("unable get events from TBA for year %d", s.Year)
		}
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	if err := s.Store.EventsUpsert(ctx, events); err != nil {
		if ctx.Err() != context.Canceled {
			s.Logger.WithError(err).Errorf("upserting events")
		}
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	if err := s.Store.MarkEventsDeleted(ctx, events); err != nil && ctx.Err() != context.Canceled {
		s.Logger.WithError(err).Errorf("marking missing events deleted")
	}
}

// updateTeams gets new team data from TBA and upserts that team data into the database.
func (s *Service) updateTeams(ctx context.Context) {
	teams, err := s.TBA.GetTeams(ctx)
	if errors.Is(err, tba.ErrNotModified{}) {
		return
	} else if err != nil {
		if ctx.Err() != context.Canceled {
			s.Logger.WithError(err).Errorf("retrieving teams")
		}
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	if err := s.Store.TeamsUpsert(ctx, teams); err != nil && ctx.Err() != context.Canceled {
		s.Logger.WithError(err).Errorf("upserting teams")
	}
}

// updateMatches gets new match data from TBA for a particular event and upserts that match data into the database.
func (s *Service) updateMatches(ctx context.Context, events []store.Event) {
	for _, event := range events {
		var fullMatches []store.Match
		var err error

		if !event.TBADeleted {
			fullMatches, err = s.TBA.GetMatches(ctx, event.Key)
			if errors.Is(err, tba.ErrNotModified{}) {
				continue
			} else if err != nil {
				if ctx.Err() != context.Canceled {
					s.Logger.WithError(err).Errorf("unable to fetch matches from TBA")
				}
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := s.Store.UpdateTBAMatches(ctx, event.Key, fullMatches); err != nil {
				if ctx.Err() != context.Canceled {
					s.Logger.WithError(err).Errorf("unable to update matches")
				}
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}

		err = s.Store.MarkMatchesDeleted(ctx, event.Key, fullMatches)
		if err != nil {
			if ctx.Err() != context.Canceled {
				s.Logger.WithError(err).Errorf("unable to mark deleted matches")
			}
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// updateEventTeamRankings gets new team rankings data from TBA for a particular event and upserts that data into the database.
func (s *Service) updateEventTeamRankings(ctx context.Context, events []store.Event) {
	for _, event := range events {
		teams, err := s.TBA.GetTeamRankings(ctx, event.Key)
		if errors.Is(err, tba.ErrNotModified{}) {
			continue
		} else if err != nil {
			if ctx.Err() != context.Canceled {
				s.Logger.WithError(err).Error("getting team ranking data from TBA")
			}
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := s.Store.EventTeamsUpsert(ctx, teams); err != nil {
			if ctx.Err() != context.Canceled {
				s.Logger.WithError(err).Error("upserting team ranking data")
			}
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
