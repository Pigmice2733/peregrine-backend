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

		if r.Method == "OPTIONS" {
			return
		}

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
	length int
	code   int
	http.ResponseWriter
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.code == 0 {
		r.code = http.StatusOK
	}

	n, err := r.ResponseWriter.Write(b)
	r.length += n
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

		roles := GetRoles(r)

		fields := logrus.Fields{
			"method":      r.Method,
			"remoteAddr":  r.RemoteAddr,
			"url":         r.URL.String(),
			"startTime":   start.Unix(),
			"requestTime": end.Sub(start).Seconds(),
			"statusCode":  rr.code,
			"bodySize":    rr.length,
			"admin":       roles.IsAdmin,
			"superAdmin":  roles.IsSuperAdmin,
			"userAgent":   r.Header.Get("User-Agent"),
		}

		if sub, err := GetSubject(r); err == nil {
			fields["userId"] = sub
		}

		if realm, err := GetRealmID(r); err == nil {
			fields["realmId"] = realm
		}

		withFields := l.WithFields(fields)
		if rr.code >= 200 && rr.code < 300 {
			withFields.Info("got request")
		} else if rr.code < 500 {
			withFields.Warn("got request")
		} else {
			withFields.Error("got request")
		}
	}
}

// Auth returns a middleware used for jwt authentication.
func Auth(next http.Handler, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			next.ServeHTTP(w, r)
			return
		}

		ss := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		token, err := jwt.ParseWithClaims(ss, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(secret), nil
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
		ctx = context.WithValue(ctx, keyRealmContext, claims.RealmID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ACL returns a middleware that must be used inside of an Auth middleware for
// checking user roles. The SuperOrAdmin requirement will be satisfied by any
// user who is either a SuperAdmin or a realm Admin.
func ACL(next http.HandlerFunc, requireSuperOrAdmin, requireVerified, requireLoggedIn bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := GetSubject(r)
		if err != nil && requireLoggedIn {
			Error(w, http.StatusUnauthorized)
			return
		}

		roles := GetRoles(r)
		if (requireSuperOrAdmin && !roles.IsSuperAdmin && !roles.IsAdmin) || (requireVerified && !roles.IsVerified) {
			Error(w, http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
