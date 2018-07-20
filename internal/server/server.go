package server

import (
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	routes "github.com/Pigmice2733/peregrine-backend/internal/routes/v1"
	"github.com/gorilla/mux"
)

// Server holds information neccesary for the peregrine-backend server.
type Server struct {
	router  *mux.Router
	address string
	origin  string
}

// New returns a new peregrine server.
func New(c config.Config) *Server {
	s := &Server{address: c.Server.Address, origin: c.Server.Origin}

	r := mux.NewRouter()
	initRoutes(r)
	s.router = r

	return s
}

// Run starts serving at the given address.
func (s *Server) Run() error {
	return http.ListenAndServe(s.address, s.router)
}

func initRoutes(r *mux.Router) {
	r = r.PathPrefix("/v1").Subrouter()
	r.Use(ihttp.CORS, ihttp.JSON)

	routes.HealthRoutes(r)
}
