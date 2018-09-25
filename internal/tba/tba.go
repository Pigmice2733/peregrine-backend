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

type tbaDistrict struct {
	Abbreviation string `json:"abbreviation"`
}

type tbaWebcast struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
}

type tbaEvent struct {
	Key          string       `json:"key"`
	ShortName    string       `json:"short_name"`
	District     *tbaDistrict `json:"district"`
	Lat          float64      `json:"lat"`
	Lng          float64      `json:"lng"`
	LocationName string       `json:"location_name"`
	Week         *int         `json:"week"`
	StartDate    string       `json:"start_date"`
	EndDate      string       `json:"end_date"`
	TimeZone     string       `json:"timezone"`
	Webcasts     []tbaWebcast `json:"webcasts"`
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
		return nil, fmt.Errorf("TBA request failed: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TBA request failed with status code: %v", response.StatusCode)
	}

	var tbaEvents []tbaEvent
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseSize)).Decode(&tbaEvents); err != nil {
		return nil, err
	}

	var events []store.Event
	for _, tbaEvent := range tbaEvents {
		var district *string
		if tbaEvent.District != nil {
			district = &tbaEvent.District.Abbreviation
		}

		timeZone, err := time.LoadLocation(tbaEvent.TimeZone)
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

		events = append(events, store.Event{
			ID:        tbaEvent.Key,
			Name:      tbaEvent.ShortName,
			District:  district,
			Week:      tbaEvent.Week,
			StartDate: store.NewUnix(startDate),
			EndDate:   store.NewUnix(endDate),
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
