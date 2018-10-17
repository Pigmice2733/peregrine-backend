package tba

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
)

// Service provides an interface to retreive data from
// The Blue Alliance's API
type Service struct {
	URL    string
	APIKey string
}

type district struct {
	Abbreviation string `json:"abbreviation"`
}

type webcast struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

type event struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	ShortName    string    `json:"short_name"`
	District     *district `json:"district"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	LocationName string    `json:"location_name"`
	Week         *int      `json:"week"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	Timezone     string    `json:"timezone"`
	Webcasts     []webcast `json:"webcasts"`
}

type alliance struct {
	Score    *int     `json:"score"`
	TeamKeys []string `json:"team_keys"`
}

type match struct {
	Key           string `json:"key"`
	PredictedTime int64  `json:"predicted_time"`
	ActualTime    int64  `json:"actual_time"`
	ScheduledTime int64  `json:"time"`
	Alliances     struct {
		Red  alliance `json:"red"`
		Blue alliance `json:"blue"`
	} `json:"alliances"`
}

type rankings struct {
	Rankings      []rank          `json:"rankings"`
	SortOrderInfo []sortOrderInfo `json:"sort_order_info"`
}

type rank struct {
	Rank       int       `json:"rank"`
	TeamKey    string    `json:"team_key"`
	SortOrders []float64 `json:"sort_orders"`
}

type sortOrderInfo struct {
	Name string `json:"name"`
}

// Maximum size of response from the TBA API to read. This value is about 4x the
// size of a typical /events/{year} response from TBA.
const maxResponseSize int64 = 1.2e+6

var tbaClient = &http.Client{
	Timeout: time.Second * 10,
}

func (s *Service) makeRequest(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", s.URL+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-TBA-Auth-Key", s.APIKey)

	return tbaClient.Do(req)
}

func webcastURL(webcastType store.WebcastType, channel string) (string, error) {
	switch webcastType {
	case store.Twitch:
		return fmt.Sprintf("https://www.twitch.tv/%s", channel), nil
	case store.Youtube:
		return fmt.Sprintf("https://www.youtube.com/watch?v=%s", channel), nil
	}

	return "", fmt.Errorf("got invalid webcast url")
}

// GetEvents retreives all events from the given year (e.g. 2018).
func (s *Service) GetEvents(year int) ([]store.Event, error) {
	path := fmt.Sprintf("/events/%d", year)

	response, err := s.makeRequest(path)
	if err != nil {
		return nil, fmt.Errorf("TBA request to %s failed: %v", path, err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TBA request to %s failed with status code: %v", path, response.StatusCode)
	}

	var tbaEvents []event
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseSize)).Decode(&tbaEvents); err != nil {
		return nil, err
	}

	var events []store.Event
	for _, tbaEvent := range tbaEvents {
		var district *string
		if tbaEvent.District != nil {
			district = &tbaEvent.District.Abbreviation
		}

		timeZone, err := time.LoadLocation(tbaEvent.Timezone)
		if err != nil {
			return nil, err
		}

		startDate, err := time.ParseInLocation("2006-01-02", tbaEvent.StartDate, timeZone)
		if err != nil {
			return nil, err
		}
		endDate, err := time.ParseInLocation("2006-01-02", tbaEvent.EndDate, timeZone)
		if err != nil {
			return nil, err
		}

		var webcasts []store.Webcast
		for _, webcast := range tbaEvent.Webcasts {
			wt := store.WebcastType(webcast.Type)
			url, err := webcastURL(wt, webcast.Channel)
			if err == nil {
				webcasts = append(webcasts, store.Webcast{Type: wt, URL: url})
			}
		}

		name := tbaEvent.ShortName
		if name == "" {
			name = tbaEvent.Name
		}

		events = append(events, store.Event{
			Key:       tbaEvent.Key,
			Name:      name,
			District:  district,
			Week:      tbaEvent.Week,
			StartDate: store.NewUnixFromTime(startDate),
			EndDate:   store.NewUnixFromTime(endDate),
			Webcasts:  webcasts,
			Location: store.Location{
				Lat:  tbaEvent.Lat,
				Lon:  tbaEvent.Lng,
				Name: tbaEvent.LocationName,
			},
		})
	}

	return events, nil
}

// GetMatches retrieves all matches from a specific event.
func (s *Service) GetMatches(eventKey string) ([]store.Match, error) {
	path := fmt.Sprintf("/event/%s/matches/simple", eventKey)

	response, err := s.makeRequest(path)
	if err != nil {
		return nil, fmt.Errorf("TBA request to %s failed: %v", path, err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TBA request to %s failed with status code: %v", path, response.StatusCode)
	}

	var tbaMatches []match
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseSize)).Decode(&tbaMatches); err != nil {
		return nil, err
	}

	var matches []store.Match
	for _, tbaMatch := range tbaMatches {
		var predictedTime *store.UnixTime
		var actualTime *store.UnixTime
		var scheduledTime *store.UnixTime

		if tbaMatch.PredictedTime != 0 {
			timestamp := store.NewUnixFromInt(tbaMatch.PredictedTime)
			predictedTime = &timestamp
		}

		if tbaMatch.ActualTime != 0 {
			timestamp := store.NewUnixFromInt(int64(tbaMatch.ActualTime))
			actualTime = &timestamp
		}

		if tbaMatch.ScheduledTime != 0 {
			timestamp := store.NewUnixFromInt(int64(tbaMatch.ScheduledTime))
			scheduledTime = &timestamp
		}

		var redScore, blueScore *int
		if tbaMatch.Alliances.Red.Score != nil && *tbaMatch.Alliances.Red.Score != -1 {
			redScore = tbaMatch.Alliances.Red.Score
		}
		if tbaMatch.Alliances.Blue.Score != nil && *tbaMatch.Alliances.Blue.Score != -1 {
			blueScore = tbaMatch.Alliances.Blue.Score
		}

		match := store.Match{
			Key:           tbaMatch.Key,
			EventKey:      eventKey,
			PredictedTime: predictedTime,
			ActualTime:    actualTime,
			ScheduledTime: scheduledTime,
			RedScore:      redScore,
			BlueScore:     blueScore,
			RedAlliance:   tbaMatch.Alliances.Red.TeamKeys,
			BlueAlliance:  tbaMatch.Alliances.Blue.TeamKeys,
		}
		matches = append(matches, match)
	}

	return matches, nil
}

// GetTeamKeys retrieves all team keys from a specific event
func (s *Service) GetTeamKeys(eventKey string) ([]string, error) {
	path := fmt.Sprintf("/event/%s/teams/keys", eventKey)

	response, err := s.makeRequest(path)
	if err != nil {
		return nil, fmt.Errorf("TBA request to %s failed: %v", path, err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TBA request to %s failed with status code: %v", path, response.StatusCode)
	}

	var teamKeys []string
	err = json.NewDecoder(io.LimitReader(response.Body, maxResponseSize)).Decode(&teamKeys)
	return teamKeys, err
}

// GetTeamRankings retrieves all team rankings from a specific event.
func (s *Service) GetTeamRankings(eventKey string) ([]store.Team, error) {
	path := fmt.Sprintf("/event/%s/rankings", eventKey)

	response, err := s.makeRequest(path)
	if err != nil {
		return nil, fmt.Errorf("TBA request to %s failed: %v", path, err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TBA request to %s failed with status code: %v", path, response.StatusCode)
	}

	teamRankings := rankings{
		Rankings:      []rank{},
		SortOrderInfo: []sortOrderInfo{},
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseSize)).Decode(&teamRankings); err != nil {
		return nil, err
	}

	rankingScoreIndex := -1
	for i, sortOrder := range teamRankings.SortOrderInfo {
		if sortOrder.Name == "Ranking Score" {
			rankingScoreIndex = i
			break
		}
	}

	var teams []store.Team
	for _, teamRank := range teamRankings.Rankings {
		var rankingScore *float64
		if rankingScoreIndex != -1 {
			rankingScore = &teamRank.SortOrders[rankingScoreIndex]
		}

		rank := teamRank.Rank
		team := store.Team{
			Key:          teamRank.TeamKey,
			EventKey:     eventKey,
			Rank:         &rank,
			RankingScore: rankingScore,
		}
		teams = append(teams, team)
	}

	return teams, nil
}
