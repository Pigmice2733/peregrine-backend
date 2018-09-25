package tba

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
)

type tbaServer struct {
	*httptest.Server
	getEventsHandler func(w http.ResponseWriter, r *http.Request)
}

const testingYear = 2018

func newInt(a int) *int {
	return &a
}

func strPointer(s string) *string {
	return &s
}

func newTBAServer() *tbaServer {
	ts := new(tbaServer)

	mux := http.NewServeMux()

	mux.HandleFunc("/events/"+strconv.Itoa(testingYear), func(w http.ResponseWriter, r *http.Request) { ts.getEventsHandler(w, r) })

	ts.Server = httptest.NewServer(mux)

	return ts
}

func TestGetEvents(t *testing.T) {
	server := newTBAServer()
	defer server.Close()

	s := Service{URL: server.URL, APIKey: "notARealKey"}

	testCases := []struct {
		getEventsHandler func(w http.ResponseWriter, r *http.Request)
		events           []store.Event
		expectErr        bool
	}{
		{
			getEventsHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			events:    nil,
			expectErr: true,
		},
		{
			getEventsHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != "notARealKey" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				[
                    {
						"key": "key1",
						"short_name": "event1",
						"district": null,
						"week": null,
						"start_date": "2018-04-02",
						"end_date": "2018-04-04",
						"webcasts": [
							{
								"channel": "nefirst_blue",
								"type": "twitch"					
							}
						],
						"lat": 41.9911025,
						"lng": -70.993044,
						"location_name": "location1",
						"timezone": "America/Los_Angeles"
					}
				]
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			events: []store.Event{
				{
					ID:        "key1",
					Name:      "event1",
					Week:      nil,
					District:  nil,
					StartDate: store.NewUnix(time.Date(2018, 4, 2, 7, 0, 0, 0, time.UTC)),
					EndDate:   store.NewUnix(time.Date(2018, 4, 4, 7, 0, 0, 0, time.UTC)),
					Location: store.Location{
						Lat:  41.9911025,
						Lon:  -70.993044,
						Name: "location1",
					},
					Webcasts: []store.Webcast{{
						Type: store.Twitch,
						URL:  "https://www.twitch.tv/nefirst_blue",
					}},
				},
			},
			expectErr: false,
		},
		{
			getEventsHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != "notARealKey" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				[
                    {
						"key": "key2",
						"short_name": "Event",
						"district": {
							"abbreviation": "ABC",
							"display_name": "Display name, not abbreviation",
							"key": "a_key",
							"year": 2018
						},
						"week": 5,
						"start_date": "2018-05-06",
						"end_date": "2018-05-07",
						"webcasts": [{
							"channel": "rXP6Vz9-Jjg",
							"type": "youtube"
						}, {
							"channel": "firstinspires12",
							"type": "twitch"
						}
					    ],
						"lat": 42.0,
						"lng": 0.0,
						"location_name": "answer"
					},
					{
						"key": "key3",
						"short_name": "PIGMICE_IS_BEST",
						"district": {
							"abbreviation": "PNW",
							"display_name": "Display name, not abbreviation",
							"key": "a_key",
							"year": 2018
						},
						"week": 2,
						"start_date": "2018-11-19",
						"end_date": "2018-11-23",
						"webcasts": [
							{
								"channel": "fakeIFRAME",
								"type": "iframe"									
							},
							{
								"channel": "gmsHpsSavuc",
								"type": "youtube"									
							}],
						"lat": 45.52,
						"lng": -122.681944,
						"location_name": "Portland",
						"timezone": "America/Los_Angeles"
				    }
				]
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			events: []store.Event{{
				ID:        "key2",
				Name:      "Event",
				District:  strPointer("ABC"),
				Week:      newInt(5),
				StartDate: store.NewUnix(time.Date(2018, 5, 6, 0, 0, 0, 0, time.UTC)),
				EndDate:   store.NewUnix(time.Date(2018, 5, 7, 0, 0, 0, 0, time.UTC)),
				Location: store.Location{
					Lat:  42.0,
					Lon:  0.0,
					Name: "answer",
				},
				Webcasts: []store.Webcast{{
					Type: store.Youtube,
					URL:  "https://www.youtube.com/watch?v=rXP6Vz9-Jjg",
				}, {
					Type: store.Twitch,
					URL:  "https://www.twitch.tv/firstinspires12",
				}},
			}, {
				ID:        "key3",
				Name:      "PIGMICE_IS_BEST",
				District:  strPointer("PNW"),
				Week:      newInt(2),
				StartDate: store.NewUnix(time.Date(2018, 11, 19, 8, 0, 0, 0, time.UTC)),
				EndDate:   store.NewUnix(time.Date(2018, 11, 23, 8, 0, 0, 0, time.UTC)),
				Location: store.Location{
					Lat:  45.52,
					Lon:  -122.681944,
					Name: "Portland",
				},
				Webcasts: []store.Webcast{{
					Type: store.Youtube,
					URL:  "https://www.youtube.com/watch?v=gmsHpsSavuc",
				}},
			}},
			expectErr: false,
		},
	}

	for index, tt := range testCases {
		server.getEventsHandler = tt.getEventsHandler

		events, err := s.GetEvents(testingYear)
		if tt.expectErr != (err != nil) {
			t.Errorf("test #%v - got error: %v, expected error: %v", index+1, err, tt.expectErr)
		}

		if !reflect.DeepEqual(events, tt.events) {
			t.Errorf("test #%v - got events: %#v\n    expected: %#v", index+1, events, tt.events)
		}
	}
}
