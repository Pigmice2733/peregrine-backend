package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	jwt "github.com/dgrijalva/jwt-go"

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
	jwtSecret        []byte
	year             int
	logger           *log.Logger
	eventsLastUpdate *time.Time
}

// New creates a new Peregrine API server
func New(tba tba.Service, store store.Service, httpAddress, httpsAddress, certFile, keyFile, origin string, jwtSecret []byte, year int) Server {
	s := Server{
		tba:          tba,
		store:        store,
		httpAddress:  httpAddress,
		httpsAddress: httpsAddress,
		certFile:     certFile,
		keyFile:      keyFile,
		year:         year,
		jwtSecret:    jwtSecret,
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

type claims struct {
	Roles []string `json:"roles"`
	jwt.StandardClaims
}

const (
	// adminRole defines the role for an administrator user.
	adminRole = "admin"
	// verifiedRole defines the role for a user that has a verified account.
	verifiedRole = "verified"
)

func (s *Server) authMiddleware(next http.HandlerFunc, optional, requireAdmin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if optional && r.Header.Get("Authentication") == "" {
			next(w, r)
			return
		}

		ss := strings.TrimPrefix(r.Header.Get("Authentication"), "Bearer ")
		token, err := jwt.ParseWithClaims(ss, &claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return s.jwtSecret, nil
		})
		if err != nil {
			s.logger.Printf("Error: parsing jwt: %v\n", err)
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		if !token.Valid {
			s.logger.Printf("Error: got invalid token: %v\n", err)
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		claims, ok := token.Claims.(*claims)
		if !ok {
			s.logger.Printf("Error: got incorrect claims type: %v\n", err)
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		if requireAdmin && !contains(claims.Roles, adminRole) {
			ihttp.Error(w, http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), keyRolesContext, claims.Roles)
		ctx = context.WithValue(ctx, keySubjectContext, claims.Subject)

		next(w, r.WithContext(ctx))
	}
}
