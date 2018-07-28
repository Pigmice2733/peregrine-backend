package http

import (
	"net/http"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// CORS is a middleware for setting Cross Origin Resource Sharing headers.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Correlation-ID")

		next.ServeHTTP(w, r)
	})
}

// MakeLoggerMiddleware returns a middleware that logs request info.
func MakeLoggerMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			correlationID := r.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				if correlationUUID, err := uuid.NewV4(); err == nil {
					correlationID = correlationUUID.String()
				}
			}

			logger.WithFields(logrus.Fields{
				"url":           r.URL.String(),
				"method":        r.Method,
				"remoteAddr":    r.RemoteAddr,
				"correlationID": correlationID,
			}).Info("Incoming HTTP request")

			next.ServeHTTP(w, r)
		})
	}
}
