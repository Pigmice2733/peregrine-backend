package server

import (
	"context"
	"net/http"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/Pigmice2733/peregrine-backend/internal/config"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/sirupsen/logrus"
)

//go:generate go run ../cmd/pack/pack.go -package server -in openapi.yaml -out openapi.go -name openAPI
var openAPI []byte

// Server is the scouting API server
type Server struct {
	config.Server

	TBA              tba.Service
	Store            store.Service
	Logger           *logrus.Logger
	eventsLastUpdate *time.Time
	start            time.Time
}

// Run starts the server, and returns if it runs into an error
func (s *Server) Run() error {
	router := s.registerRoutes()

	var handler http.Handler = router
	handler = ihttp.LimitBody(handler)
	handler = gziphandler.GzipHandler(handler)
	handler = ihttp.Log(handler, s.Logger)
	handler = ihttp.Auth(handler, s.JWTSecret)
	handler = ihttp.CORS(handler, s.Origin)

	s.Logger.Info("fetching seed events")
	if err := s.updateEvents(context.Background()); err != nil {
		s.Logger.WithError(err).Error("updating event data on server run")
	}
	s.Logger.Info("fetched seed events")

	httpServer := &http.Server{
		Addr:              s.Listen,
		Handler:           handler,
		ReadTimeout:       time.Second * 15,
		ReadHeaderTimeout: time.Second * 15,
		WriteTimeout:      time.Second * 15,
		IdleTimeout:       time.Second * 30,
		MaxHeaderBytes:    4096,
	}

	s.start = time.Now()
	s.Logger.WithField("httpAddress", s.Listen).Info("serving http")
	return httpServer.ListenAndServe()
}

type services struct {
	TBA        bool `json:"tba"`
	PostgreSQL bool `json:"postgresql"`
}

type status struct {
	StartTime int64    `json:"startTime"`
	Uptime    int64    `json:"uptime"`
	Listen    string   `json:"listen"`
	Services  services `json:"services"`
	Ok        bool     `json:"ok"`
}

func (s *Server) openAPIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(openAPI)
	}
}

func (s *Server) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tbaHealthy := s.TBA.Ping(r.Context()) == nil
		pgHealthy := s.Store.Ping(r.Context()) == nil

		ihttp.Respond(w, status{
			StartTime: s.start.Unix(),
			Uptime:    int64(time.Since(s.start).Seconds()),
			Listen:    s.Listen, //todo update swagger
			Services: services{
				TBA:        tbaHealthy,
				PostgreSQL: pgHealthy,
			},
			Ok: tbaHealthy && pgHealthy,
		}, http.StatusOK)
	}
}
