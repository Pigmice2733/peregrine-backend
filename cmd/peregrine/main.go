package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/Pigmice2733/peregrine-backend/internal/server"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func main() {
	var basePath = flag.String("basePath", ".", "Path to the etc directory where the config file is.")

	flag.Parse()

	if err := run(*basePath); err != nil {
		fmt.Printf("got error: %v\n", err)
		os.Exit(1)
	}
}

func run(basePath string) error {
	c, err := config.Open(basePath)
	if err != nil {
		return errors.Wrap(err, "opening config")
	}

	tba := tba.Service{
		URL:    c.TBA.URL,
		APIKey: c.TBA.APIKey,
	}

	logger := logrus.New()
	if c.Server.LogJSON {
		logger.Formatter = &logrus.JSONFormatter{}
	}

	sto, err := store.New(context.Background(), c.DSN)
	if err != nil {
		return errors.Wrap(err, "opening postgres server")
	}
	defer sto.Close()

	if c.Server.Year == 0 {
		c.Server.Year = time.Now().Year()
	}

	jwtSecret := make([]byte, 64)
	if _, err := rand.Read(jwtSecret); err != nil {
		return errors.Wrap(err, "generating jwt secret")
	}

	logger := logrus.New()
	if c.Server.LogJSON {
		logger.Formatter = &logrus.JSONFormatter{}
	}

	s := &server.Server{
		TBA:       tba,
		Store:     sto,
		Logger:    logger,
		Server:    c.Server,
		JWTSecret: jwtSecret,
	}

	return errors.Wrap(s.Run(context.Background()), "running server")
}
