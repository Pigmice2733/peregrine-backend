package routes

import (
	"github.com/Pigmice2733/peregrine-backend/internal/handlers"
	"github.com/gorilla/mux"
)

// HealthRoutes registers all the routes for health handlers.
func HealthRoutes(r *mux.Router) {
	r.HandleFunc("/health", handlers.Health()).Methods("GET")
}
