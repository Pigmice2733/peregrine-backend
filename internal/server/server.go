package server

import (
	"log"
	"net/http"
	"os"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
)

// Server is the scouting API server
type Server struct {
	tba     tba.Service
	handler http.Handler
	address string
	year    int
	logger  *log.Logger
}

// New creates a new Peregrine API server
func New(tba tba.Service, address string, year int) Server {
	s := Server{
		tba:     tba,
		address: address,
		year:    year,
	}

	router := s.registerRoutes()
	s.handler = ihttp.CORS(ihttp.LimitBody(router))

	s.logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	return s
}

// Run starts the server, and returns if it runs into an error
func (s *Server) Run() error {
	return http.ListenAndServe(s.address, s.handler)
}
