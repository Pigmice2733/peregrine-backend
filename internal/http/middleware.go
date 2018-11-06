package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

// CORS is a middleware for setting Cross Origin Resource Sharing headers.
func CORS(next http.Handler, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		next.ServeHTTP(w, r)
	})
}

// LimitBody is middleware to protect the server from requests containing
// massive amounts of data.
func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1000000) // 1 MB
		next.ServeHTTP(w, r)
	})
}

type recorder struct {
	len  int
	code int
	http.ResponseWriter
}

func (r *recorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.len += n
	return n, err
}

func (r *recorder) WriteHeader(statusCode int) {
	r.code = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Log logs information about incoming HTTP requests.
func Log(next http.Handler, l *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rr := &recorder{ResponseWriter: w}

		start := time.Now()
		next.ServeHTTP(rr, r)
		end := time.Now()

		fields := logrus.Fields{
			"method":       r.Method,
			"remote_addr":  r.RemoteAddr,
			"url":          r.URL.String(),
			"start_time":   start.Unix(),
			"request_time": end.Sub(start).Seconds(),
			"status_code":  rr.code,
			"body_size":    rr.len,
			"admin":        GetRoles(r).IsAdmin,
		}

		if sub, err := GetSubject(r); err != nil {
			fields["user_id"] = sub
		}

		l.WithFields(fields).Info("got request")
	}
}

// Auth returns a middleware used for jwt authentication.
func Auth(next http.HandlerFunc, jwtSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authentication") == "" {
			next(w, r)
			return
		}

		ss := strings.TrimPrefix(r.Header.Get("Authentication"), "Bearer ")
		token, err := jwt.ParseWithClaims(ss, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return jwtSecret, nil
		})
		if err != nil {
			Error(w, http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			Error(w, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			Error(w, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), keyRolesContext, claims.Roles)
		ctx = context.WithValue(ctx, keySubjectContext, claims.Subject)

		next(w, r.WithContext(ctx))
	}
}

// ACL returns a middleware that must be used inside of an Auth middleware for
// checking user roles.
func ACL(next http.HandlerFunc, requireAdmin, requireVerified bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roles := GetRoles(r)

		if (requireAdmin && !roles.IsAdmin) || (requireVerified && !roles.IsVerified) {
			Error(w, http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
