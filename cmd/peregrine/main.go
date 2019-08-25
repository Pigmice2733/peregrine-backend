package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/Pigmice2733/peregrine-backend/internal/server"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
	"github.com/Pigmice2733/peregrine-backend/internal/tbaupdater"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [config path]\n", os.Args[0])
	}

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	if err := run(args[0]); err != nil {
		fmt.Printf("got error: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	c, err := config.Open(configPath)
	if err != nil {
		return errors.Wrap(err, "unable to open config")
	}

	tba := &tba.Service{
		URL:    c.TBA.URL,
		APIKey: c.TBA.APIKey,
	}

	logger := logrus.New()
	logger.SetLevel(c.Server.LogLevel)
	if c.Server.LogJSON {
		logger.Formatter = &logrus.JSONFormatter{}
	}

	logger.Info("connecting to postgres")
	sto, err := store.New(context.Background(), c.DSN, logger)
	if err != nil {
		return errors.Wrap(err, "opening postgres server")
	}
	defer sto.Close()
	logger.Info("connected to postgres")

	tbaUpdates := &tbaupdater.Service{
		TBA:    tba,
		Store:  sto,
		Logger: logger,
		Year:   c.Year,
	}

	s := &server.Server{
		TBA:    tba,
		Store:  sto,
		Logger: logger,
		Server: c.Server,
	}

	tbaUpdates.Begin()
	err = errors.Wrap(s.Run(), "running server")
	tbaUpdates.End()

	return err
}
