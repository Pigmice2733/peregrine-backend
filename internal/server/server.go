package server

import (
	"net/http"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Server holds information neccesary for the peregrine-backend server.
type Server struct {
	router    *mux.Router
	startTime *time.Time
	logger    *logrus.Logger
	address   string
	origin    string
}

// New returns a new peregrine server.
func New(c config.Config, logger *logrus.Logger) *Server {
	s := &Server{address: c.Server.Address, origin: c.Server.Origin, logger: logger}

	s.initRoutesV1()

	return s
}

// Run starts serving at the given address.
func (s *Server) Run() error {
	now := time.Now()
	s.startTime = &now

	err := http.ListenAndServe(s.address, ihttp.MakeLoggerMiddleware(s.logger)(s.router))
	s.startTime = nil

	return err
}
