package server

import (
	"io"
	"net/http"
	"time"

	"github.com/NYTimes/gziphandler"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/sirupsen/logrus"
)

// Server is the scouting API server
type Server struct {
	tba              tba.Service
	store            store.Service
	handler          http.Handler
	httpAddress      string
	httpsAddress     string
	certFile         string
	keyFile          string
	jwtSecret        []byte
	year             int
	logger           *logrus.Logger
	eventsLastUpdate *time.Time
}

// New creates a new Peregrine API server
func New(tba tba.Service, store store.Service, logWriter io.Writer, logJSON bool, httpAddress, httpsAddress, certFile, keyFile, origin string, jwtSecret []byte, year int) Server {
	s := Server{
		tba:          tba,
		store:        store,
		httpAddress:  httpAddress,
		httpsAddress: httpsAddress,
		certFile:     certFile,
		keyFile:      keyFile,
		year:         year,
		jwtSecret:    jwtSecret,
	}

	s.logger = logrus.New()
	s.logger.Out = logWriter

	if logJSON {
		s.logger.Formatter = &logrus.JSONFormatter{}
	}

	router := s.registerRoutes()
	s.handler = ihttp.Auth(ihttp.Log(gziphandler.GzipHandler(ihttp.CORS(ihttp.LimitBody(router), origin)), s.logger), s.jwtSecret)

	return s
}

// Run starts the server, and returns if it runs into an error
func (s *Server) Run() error {
	s.logger.Info("fetching seed events")
	if err := s.updateEvents(); err != nil {
		s.logger.WithError(err).Error("updating event data on server run")
	}

	httpServer := &http.Server{
		Addr:              s.httpAddress,
		Handler:           s.handler,
		ReadTimeout:       time.Second * 15,
		ReadHeaderTimeout: time.Second * 15,
		WriteTimeout:      time.Second * 15,
		IdleTimeout:       time.Second * 30,
		MaxHeaderBytes:    4096,
	}
	defer httpServer.Close()

	errs := make(chan error)

	go func() {
		s.logger.WithField("http_address", s.httpAddress).Info("serving http")
		errs <- httpServer.ListenAndServe()
	}()

	if s.certFile != "" && s.keyFile != "" {
		httpsServer := &http.Server{
			Addr:              s.httpsAddress,
			Handler:           s.handler,
			ReadTimeout:       time.Second * 15,
			ReadHeaderTimeout: time.Second * 15,
			WriteTimeout:      time.Second * 15,
			IdleTimeout:       time.Second * 30,
			MaxHeaderBytes:    4096,
		}
		defer httpsServer.Close()

		go func() {
			s.logger.WithField("https_address", s.httpsAddress).Info("serving https")
			errs <- httpsServer.ListenAndServeTLS(s.certFile, s.keyFile)
		}()
	}

	return <-errs
}
