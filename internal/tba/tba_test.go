package tba

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
)

type tbaServer struct {
	*httptest.Server
	getEventsHandler       func(w http.ResponseWriter, r *http.Request)
	getMatchesHandler      func(w http.ResponseWriter, r *http.Request)
	getTeamRankingsHandler func(w http.ResponseWriter, r *http.Request)
	getTeamsHandler        func(w http.ResponseWriter, r *http.Request)
}

const testingYear = 2018

func newInt(a int) *int {
	return &a
}

func newFloat64(f float64) *float64 {
	return &f
}

func newString(s string) *string {
	return &s
}

func newTime(t time.Time) *time.Time {
	return &t
}

func newTBAServer() *tbaServer {
	ts := new(tbaServer)

	r := mux.NewRouter()
	r.HandleFunc("/events/"+strconv.Itoa(testingYear), func(w http.ResponseWriter, r *http.Request) { ts.getEventsHandler(w, r) })
	r.HandleFunc("/event/{eventKey}/matches", func(w http.ResponseWriter, r *http.Request) { ts.getMatchesHandler(w, r) })
	r.HandleFunc("/event/{eventKey}/rankings", func(w http.ResponseWriter, r *http.Request) { ts.getTeamRankingsHandler(w, r) })
	r.HandleFunc("/teams/{page}", func(w http.ResponseWriter, r *http.Request) { ts.getTeamsHandler(w, r) })

	ts.Server = httptest.NewServer(r)

	return ts
}

func TestGetEvents(t *testing.T) {
	server := newTBAServer()
	defer server.Close()

	APIKey := "notARealKey"

	s := Service{URL: server.URL, APIKey: APIKey}

	testCases := []struct {
		name             string
		getEventsHandler func(w http.ResponseWriter, r *http.Request)
		events           []store.Event
		expectErr        bool
	}{
		{
			name: "tba events route gives 500",
			getEventsHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			events:    nil,
			expectErr: true,
		},
		{
			name: "tba gives non-nullable event data only",
			getEventsHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != APIKey {
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
						"gmaps_url": "https://www.google.com/maps?cid=7437893320196269298",
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
					Key:          "key1",
					Name:         "event1",
					Week:         nil,
					District:     nil,
					FullDistrict: nil,
					StartDate:    time.Date(2018, 4, 2, 7, 0, 0, 0, time.UTC),
					EndDate:      time.Date(2018, 4, 4, 7, 0, 0, 0, time.UTC),
					Lat:          41.9911025,
					Lon:          -70.993044,
					GMapsURL:     newString("https://www.google.com/maps?cid=7437893320196269298"),
					LocationName: "location1",
					Webcasts:     []string{"https://www.twitch.tv/nefirst_blue"},
				},
			},
			expectErr: false,
		},
		{
			name: "tba gives full event data",
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
							"display_name": "Full ABC",
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
						"gmaps_url": "https://www.google.com/maps?cid=7437893320196269298",
						"location_name": "answer"
					},
					{
						"key": "key3",
						"name": "PIGMICE_IS_BEST",
						"short_name": "",
						"district": {
							"abbreviation": "PNW",
							"display_name": "Full PNW",
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
						"gmaps_url": "https://www.google.com/maps?cid=7437893320196269298",
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
				Key:          "key2",
				Name:         "Event",
				District:     newString("ABC"),
				FullDistrict: newString("Full ABC"),
				Week:         newInt(5),
				StartDate:    time.Date(2018, 5, 6, 0, 0, 0, 0, time.UTC),
				EndDate:      time.Date(2018, 5, 7, 0, 0, 0, 0, time.UTC),
				Lat:          42.0,
				Lon:          0.0,
				GMapsURL:     newString("https://www.google.com/maps?cid=7437893320196269298"),
				LocationName: "answer",
				Webcasts:     []string{"https://www.youtube.com/watch?v=rXP6Vz9-Jjg", "https://www.twitch.tv/firstinspires12"},
			}, {
				Key:          "key3",
				Name:         "PIGMICE_IS_BEST",
				District:     newString("PNW"),
				FullDistrict: newString("Full PNW"),
				Week:         newInt(2),
				StartDate:    time.Date(2018, 11, 19, 8, 0, 0, 0, time.UTC),
				EndDate:      time.Date(2018, 11, 23, 8, 0, 0, 0, time.UTC),
				Lat:          45.52,
				Lon:          -122.681944,
				GMapsURL:     newString("https://www.google.com/maps?cid=7437893320196269298"),
				LocationName: "Portland",
				Webcasts:     []string{"https://www.youtube.com/watch?v=gmsHpsSavuc"},
			}},
			expectErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			server.getEventsHandler = tt.getEventsHandler

			events, err := s.GetEvents(context.TODO(), testingYear)
			if !tt.expectErr && err != nil {
				t.Errorf("did not expect an error but got one: %v", err)
			} else if tt.expectErr && err == nil {
				t.Errorf("expected error but didnt get one: %v", err)
			}

			if !cmp.Equal(events, tt.events) {
				t.Errorf("expected events do not equal actual events, got dif: %s", cmp.Diff(tt.events, events))
			}
		})
	}
}

func TestGetMatches(t *testing.T) {
	server := newTBAServer()
	defer server.Close()

	APIKey := "alsoNotARealKey"

	s := Service{URL: server.URL, APIKey: APIKey}

	const eventKey = "2018alhu"

	testCases := []struct {
		name              string
		getMatchesHandler func(w http.ResponseWriter, r *http.Request)
		matches           []store.Match
		expectErr         bool
	}{
		{
			name: "tba matches route gives 500",
			getMatchesHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			matches:   nil,
			expectErr: true,
		},
		{
			name: "tba gives full matches data",
			getMatchesHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != APIKey {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				vars := mux.Vars(r)
				if key, ok := vars["eventKey"]; !ok || key != eventKey {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				[
                    {
						"key": "event_key1",
						"alliances": {
							"red": {
								"score": 220,
								"team_keys": ["frc254", "frc1234", "frc00"]
							},
							"blue": {
								"score": 500,
								"team_keys": ["frc2733", "frc9876", "frc1"]
							}
						},
						"score_breakdown": {
							"red": {
								"foobar": 3,
								"barbaz": true
							},
							"blue": {
								"foobar": 1,
								"blabla": 3.212
							}
						},
						"predicted_time": 1520272800,
						"time": 1520272800,
						"actual_time": 1520274000
					},
					{
						"key": "event_key2",
						"alliances": {
							"red": {
								"score": 120,
								"team_keys": ["frc0", "frc1", "frc2"]
							},
							"blue": {
								"score": 600,
								"team_keys": ["frc2", "frc7", "frc3"]
							}
						},
						"predicted_time": 1525272780,
						"time": 1525272780,
						"actual_time": 1525273980,
						"score_breakdown": {
							"red": {
								"foobar": 90.2,
								"barbaz": false
							},
							"blue": {
								"foobar": 4.2321,
								"blabla": 9
							}
						}
					}
				]
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			matches: []store.Match{
				{
					Key:                "key1",
					EventKey:           "2018alhu",
					PredictedTime:      newTime(time.Date(2018, 3, 5, 18, 0, 0, 0, time.UTC)),
					ScheduledTime:      newTime(time.Date(2018, 3, 5, 18, 0, 0, 0, time.UTC)),
					ActualTime:         newTime(time.Date(2018, 3, 5, 18, 20, 0, 0, time.UTC)),
					RedScore:           newInt(220),
					BlueScore:          newInt(500),
					RedAlliance:        []string{"frc254", "frc1234", "frc00"},
					BlueAlliance:       []string{"frc2733", "frc9876", "frc1"},
					RedScoreBreakdown:  map[string]interface{}{"foobar": 3.0, "barbaz": true},
					BlueScoreBreakdown: map[string]interface{}{"foobar": 1.0, "blabla": 3.212},
					TBAURL:             newString(tbaURL + "/match/event_key1"),
				},
				{
					Key:                "key2",
					EventKey:           "2018alhu",
					PredictedTime:      newTime(time.Date(2018, 5, 2, 14, 53, 0, 0, time.UTC)),
					ScheduledTime:      newTime(time.Date(2018, 5, 2, 14, 53, 0, 0, time.UTC)),
					ActualTime:         newTime(time.Date(2018, 5, 2, 15, 13, 0, 0, time.UTC)),
					RedScore:           newInt(120),
					BlueScore:          newInt(600),
					RedAlliance:        []string{"frc0", "frc1", "frc2"},
					BlueAlliance:       []string{"frc2", "frc7", "frc3"},
					RedScoreBreakdown:  map[string]interface{}{"foobar": 90.2, "barbaz": false},
					BlueScoreBreakdown: map[string]interface{}{"foobar": 4.2321, "blabla": 9.0},
					TBAURL:             newString(tbaURL + "/match/event_key2"),
				},
			},
			expectErr: false,
		},
		{
			name: "tba gives partial match time data",
			getMatchesHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != APIKey {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				vars := mux.Vars(r)
				if key, ok := vars["eventKey"]; !ok || key != eventKey {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				[
                    {
						"key": "event_key1",
						"alliances": {
							"red": {
								"score": -1,
								"team_keys": ["frc254", "frc1234", "frc00"]
							},
							"blue": {
								"score": -1,
								"team_keys": ["frc2733", "frc9876", "frc1"]
							}
						},
						"predicted_time": 1520272800,
						"time": 1520272800,
						"actual_time": 1520274000
					},
					{
						"key": "event_key2",
						"alliances": {
							"red": {
								"score": null,
								"team_keys": ["frc0", "frc1", "frc2"]
							},
							"blue": {
								"score": null,
								"team_keys": ["frc2", "frc7", "frc3"]
							}
						},
						"predicted_time": null,
						"time": 1525272780,
						"actual_time": null
					}
				]
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			matches: []store.Match{
				{
					Key:           "key1",
					EventKey:      "2018alhu",
					PredictedTime: newTime(time.Date(2018, 3, 5, 18, 0, 0, 0, time.UTC)),
					ScheduledTime: newTime(time.Date(2018, 3, 5, 18, 0, 0, 0, time.UTC)),
					ActualTime:    newTime(time.Date(2018, 3, 5, 18, 20, 0, 0, time.UTC)),
					RedScore:      nil,
					BlueScore:     nil,
					RedAlliance:   []string{"frc254", "frc1234", "frc00"},
					BlueAlliance:  []string{"frc2733", "frc9876", "frc1"},
					TBAURL:        newString(tbaURL + "/match/event_key1"),
				},
				{
					Key:           "key2",
					EventKey:      "2018alhu",
					PredictedTime: nil,
					ScheduledTime: newTime(time.Date(2018, 5, 2, 14, 53, 0, 0, time.UTC)),
					ActualTime:    nil,
					RedScore:      nil,
					BlueScore:     nil,
					RedAlliance:   []string{"frc0", "frc1", "frc2"},
					BlueAlliance:  []string{"frc2", "frc7", "frc3"},
					TBAURL:        newString(tbaURL + "/match/event_key2"),
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			server.getMatchesHandler = tt.getMatchesHandler

			matches, err := s.GetMatches(context.TODO(), eventKey)
			if !tt.expectErr && err != nil {
				t.Errorf("did not expect an error but got one: %v", err)
			} else if tt.expectErr && err == nil {
				t.Errorf("expected error but didnt get one: %v", err)
			}

			if !cmp.Equal(matches, tt.matches) {
				t.Errorf("expected matches do not equal actual matches, got dif: %s", cmp.Diff(tt.matches, matches))
			}
		})
	}
}

func TestGetTeams(t *testing.T) {
	server := newTBAServer()
	defer server.Close()

	const apiKey = "notARealKey"

	s := Service{URL: server.URL, APIKey: apiKey}

	testCases := []struct {
		name            string
		getTeamsHandler func(w http.ResponseWriter, r *http.Request)
		teams           []store.Team
		expectErr       bool
	}{
		{
			name: "tba teams route gives 500",
			getTeamsHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			teams:     nil,
			expectErr: true,
		},
		{
			name: "tba gives page of teams",
			getTeamsHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != apiKey {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				vars := mux.Vars(r)
				page, ok := vars["page"]
				if !ok {
					w.WriteHeader(http.StatusNotFound)
				}

				pageNum, err := strconv.Atoi(page)
				if err != nil || pageNum < 0 {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				w.WriteHeader(http.StatusOK)

				switch pageNum {
				case 0:
					_, err = w.Write([]byte(`
				[
					{
						"address": null,
						"city": "Fort Lauderdale",
						"country": "USA",
						"gmaps_place_id": null,
						"gmaps_url": null,
						"home_championship": {
							"2017": "Houston",
							"2018": "Houston"
						},
						"key": "frc7500",
						"lat": null,
						"lng": null,
						"gmaps_url": null,
						"location_name": null,
						"motto": null,
						"name": "NASA/Florida Power and Light/State of Florida&St Thomas Aquinas High School",
						"nickname": "MARAUDERS",
						"postal_code": "33312",
						"rookie_year": 2019,
						"state_prov": "Florida",
						"team_number": 7500,
						"website": null
						},
						{
						"address": null,
						"city": "Middlebury",
						"country": "USA",
						"gmaps_place_id": null,
						"gmaps_url": null,
						"home_championship": {
							"2017": "St. Louis",
							"2018": "Detroit"
						},
						"key": "frc7502",
						"lat": null,
						"lng": null,
						"gmaps_url": null,
						"location_name": null,
						"motto": null,
						"name": "NASA/Middlebury Community Schools&Northridge High School",
						"postal_code": "46540",
						"rookie_year": 2019,
						"state_prov": "Indiana",
						"team_number": 7502,
						"website": null
					}
				]
				`))
				case 1:
					_, err = w.Write([]byte(`
				[
					{
						"address": null,
						"city": "Portland",
						"country": "USA",
						"gmaps_place_id": null,
						"gmaps_url": null,
						"home_championship": {
							"2017": "Houston",
							"2018": "Houston"
						},
						"key": "frc2733",
						"lat": null,
						"lng": null,
						"gmaps_url": null,
						"location_name": null,
						"motto": null,
						"name": "Daimler/TE Connectivity/Boeing/Oregon Dept of Education/FLIR/Autodesk/DW Fritz Automation/Marathon Oil/SolidWorks/Hankins Hardware&Cleveland High School&Family/Community",
						"nickname": "Pigmice",
						"postal_code": "97202",
						"rookie_year": 2009,
						"state_prov": "Oregon",
						"team_number": 2733,
						"website": "https://www.pigmice.com"
					}
				]
				`))
				default:
					_, err = w.Write([]byte(`
				[
				]
				`))
				}

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			teams: []store.Team{
				{
					Key:      "frc7500",
					Nickname: "MARAUDERS",
				},
				{
					Key:      "frc7502",
					Nickname: "",
				},
				{
					Key:      "frc2733",
					Nickname: "Pigmice",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			server.getTeamsHandler = tt.getTeamsHandler

			teams, err := s.GetTeams(context.TODO())
			if !tt.expectErr && err != nil {
				t.Errorf("did not expect an error but got one: %v", err)
			} else if tt.expectErr && err == nil {
				t.Errorf("expected error but didnt get one: %v", err)
			}

			if !cmp.Equal(teams, tt.teams) {
				t.Errorf("expected teams do not equal actual teams, got dif: %s", cmp.Diff(tt.teams, teams))
			}
		})
	}
}

func TestGetTeamRankings(t *testing.T) {
	server := newTBAServer()
	defer server.Close()

	APIKey := "notARealKey"

	s := Service{URL: server.URL, APIKey: APIKey}

	const eventKey = "2018abca"

	testCases := []struct {
		name                   string
		getTeamRankingsHandler func(w http.ResponseWriter, r *http.Request)
		teams                  []store.EventTeam
		expectErr              bool
	}{
		{
			name: "tba rankings route gives 500",
			getTeamRankingsHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			teams:     nil,
			expectErr: true,
		},
		{
			name: "tba gives team ranking and ranking score",
			getTeamRankingsHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != APIKey {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				vars := mux.Vars(r)
				if key, ok := vars["eventKey"]; !ok || key != eventKey {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				{
				    "rankings": [
						{
							"rank": 1,
							"team_key": "frc2733",
							"sort_orders": [
								3243,
								5.25
							]
						},
						{
							"rank": 2,
							"team_key": "frc254",
							"sort_orders": [
								2453,
								2.00
							]
						}
					],
					"sort_order_info": [
						{
							"name": "Irrelevant Score",
							"precision": 0
						},
						{
							"name": "Ranking Score",
							"precision": 2
						}
					]
				}
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			teams: []store.EventTeam{
				{
					Key:          "frc2733",
					EventKey:     "2018abca",
					Rank:         newInt(1),
					RankingScore: newFloat64(5.25),
				},
				{
					Key:          "frc254",
					EventKey:     "2018abca",
					Rank:         newInt(2),
					RankingScore: newFloat64(2.00),
				},
			},
			expectErr: false,
		},
		{
			name: "tba gives only team ranking",
			getTeamRankingsHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-TBA-Auth-Key") != "notARealKey" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				vars := mux.Vars(r)
				if key, ok := vars["eventKey"]; !ok || key != eventKey {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`
				{
				    "rankings": [
						{
							"rank": 1,
							"team_key": "frc2733",
							"sort_orders": [
								3243,
								5.25000000
							]
						},
						{
							"rank": 2,
							"team_key": "frc254",
							"sort_orders": [
								23,
								2.000100
							]
						},
						{
							"rank": 12,
							"team_key": "frc24",
							"sort_orders": [
								0,
								2.000001
							]
						}
					],
					"sort_order_info": [
						{
							"name": "Irrelevant Score",
							"precision": 0
						},
						{
							"name": "Random Score",
							"precision": 12
						}
					]
				}
				`))

				if err != nil {
					t.Errorf("failed to write test data")
				}
			},
			teams: []store.EventTeam{
				{
					Key:          "frc2733",
					EventKey:     "2018abca",
					Rank:         newInt(1),
					RankingScore: nil,
				},
				{
					Key:          "frc254",
					EventKey:     "2018abca",
					Rank:         newInt(2),
					RankingScore: nil,
				},
				{
					Key:          "frc24",
					EventKey:     "2018abca",
					Rank:         newInt(12),
					RankingScore: nil,
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			server.getTeamRankingsHandler = tt.getTeamRankingsHandler

			teams, err := s.GetTeamRankings(context.TODO(), "2018abca")
			if !tt.expectErr && err != nil {
				t.Errorf("did not expect an error but got one: %v", err)
			} else if tt.expectErr && err == nil {
				t.Errorf("expected error but didnt get one: %v", err)
			}

			if !cmp.Equal(teams, tt.teams) {
				t.Errorf("expected teams do not equal actual teams, got dif: %s", cmp.Diff(tt.teams, teams))
			}
		})
	}
}
