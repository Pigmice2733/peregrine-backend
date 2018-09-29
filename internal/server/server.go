package server

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NYTimes/gziphandler"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
)

// Server is the scouting API server
type Server struct {
	tba              tba.Service
	store            store.Service
	handler          http.Handler
	address          string
	year             int
	logger           *log.Logger
	eventsLastUpdate *time.Time
}

// New creates a new Peregrine API server
func New(tba tba.Service, store store.Service, address string, origin string, year int) Server {
	s := Server{
		tba:     tba,
		store:   store,
		address: address,
		year:    year,
	}

	router := s.registerRoutes()
	s.handler = gziphandler.GzipHandler(ihttp.CORS(ihttp.LimitBody(router), origin))

	s.logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	return s
}

// Run starts the server, and returns if it runs into an error
func (s *Server) Run() error {
	s.logger.Printf("Fetching seed events")
	if err := s.updateEvents(); err != nil {
		s.logger.Printf("Error: updating event data on Run: %v\n", err)
	}

	s.logger.Printf("Listening at: %s\n", s.address)
	return http.ListenAndServe(s.address, s.handler)
}
