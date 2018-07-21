package routes

import (
	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/gorilla/mux"
)

// Register registers all v1 routes on the given router.
func Register(r *mux.Router) {
	r = r.PathPrefix("/v1").Subrouter()

	r.Use(ihttp.CORS, ihttp.JSON)

	HealthRoutes(r)
}
