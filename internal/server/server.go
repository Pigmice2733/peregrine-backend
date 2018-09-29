package server

import (
	"log"
	"net/http"
	"os"
	"time"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
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
	year             int
	logger           *log.Logger
	eventsLastUpdate *time.Time
}

// New creates a new Peregrine API server
func New(tba tba.Service, store store.Service, httpAddress, httpsAddress, certFile, keyFile, origin string, year int) Server {
	s := Server{
		tba:          tba,
		store:        store,
		httpAddress:  httpAddress,
		httpsAddress: httpsAddress,
		certFile:     certFile,
		keyFile:      keyFile,
		year:         year,
	}

	router := s.registerRoutes()
	s.handler = ihttp.CORS(ihttp.LimitBody(router), origin)

	s.logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	return s
}

// Run starts the server, and returns if it runs into an error
func (s *Server) Run() error {
	s.logger.Printf("Fetching seed events")
	if err := s.updateEvents(); err != nil {
		s.logger.Printf("Error: updating event data on Run: %v\n", err)
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
		s.logger.Printf("Serving http at: %s\n", s.httpAddress)
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
			s.logger.Printf("Serving https at: %s\n", s.httpsAddress)
			errs <- httpsServer.ListenAndServeTLS(s.certFile, s.keyFile)
		}()
	}

	return <-errs
}
