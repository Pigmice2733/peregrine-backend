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

// Server is the scouting API server
type Server struct {
	config.Server

	TBA              tba.Service
	Store            store.Service
	Logger           *logrus.Logger
	JWTSecret        []byte
	eventsLastUpdate *time.Time
	start            time.Time
}

// Run starts the server, and returns if it runs into an error
func (s *Server) Run(ctx context.Context) error {
	router := s.registerRoutes()

	var handler http.Handler = router
	handler = ihttp.LimitBody(handler)
	handler = gziphandler.GzipHandler(handler)
	handler = ihttp.Log(handler, s.Logger)
	handler = ihttp.Auth(handler, s.JWTSecret)
	handler = ihttp.CORS(handler, s.Origin)

	s.Logger.Info("fetching seed events")
	if err := s.updateEvents(ctx); err != nil {
		s.Logger.WithError(err).Error("updating event data on server run")
	}

	httpServer := &http.Server{
		Addr:              s.HTTPAddress,
		Handler:           handler,
		ReadTimeout:       time.Second * 15,
		ReadHeaderTimeout: time.Second * 15,
		WriteTimeout:      time.Second * 15,
		IdleTimeout:       time.Second * 30,
		MaxHeaderBytes:    4096,
	}
	defer httpServer.Close()

	errs := make(chan error)

	s.start = time.Now()

	go func() {
		s.Logger.WithField("http_address", s.HTTPAddress).Info("serving http")
		errs <- httpServer.ListenAndServe()
	}()

	if s.CertFile != "" && s.KeyFile != "" {
		httpsServer := &http.Server{
			Addr:              s.HTTPSAddress,
			Handler:           handler,
			ReadTimeout:       time.Second * 15,
			ReadHeaderTimeout: time.Second * 15,
			WriteTimeout:      time.Second * 15,
			IdleTimeout:       time.Second * 30,
			MaxHeaderBytes:    4096,
		}
		defer httpsServer.Close()

		go func() {
			s.Logger.WithField("https_address", s.HTTPSAddress).Info("serving https")
			errs <- httpsServer.ListenAndServeTLS(s.CertFile, s.KeyFile)
		}()
	}

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return nil
	}
}

type listen struct {
	HTTP  string `json:"http,omitempty"`
	HTTPS string `json:"https,omitempty"`
}

type services struct {
	TBA        bool `json:"tba"`
	PostgreSQL bool `json:"postgresql"`
}

type status struct {
	StartTime int64    `json:"startTime"`
	Uptime    int64    `json:"uptime"`
	Listen    listen   `json:"listen"`
	Services  services `json:"services"`
	Ok        bool     `json:"ok"`
}

func (s *Server) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tbaHealthy := s.TBA.Ping(r.Context()) == nil
		pgHealthy := s.Store.Ping(r.Context()) == nil

		ihttp.Respond(w, status{
			StartTime: s.start.Unix(),
			Uptime:    int64(time.Since(s.start).Seconds()),
			Listen: listen{
				HTTP:  s.HTTPAddress,
				HTTPS: s.HTTPSAddress,
			},
			Services: services{
				TBA:        tbaHealthy,
				PostgreSQL: pgHealthy,
			},
			Ok: tbaHealthy && pgHealthy,
		}, http.StatusOK)
	}
}
