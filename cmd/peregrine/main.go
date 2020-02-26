package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/Pigmice2733/peregrine-backend/internal/refresh"
	"github.com/Pigmice2733/peregrine-backend/internal/server"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"
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

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range c {
			cancel()
		}
	}()

	if err := run(ctx, args[0]); err != nil {
		fmt.Printf("got error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath string) error {
	c, err := config.Open(configPath)
	if err != nil {
		return fmt.Errorf("unable to open config: %w", err)
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
	sto, err := store.New(ctx, c.DSN, logger)
	if err != nil {
		return fmt.Errorf("opening postgres server: %w", err)
	}
	defer sto.Close()
	logger.Info("connected to postgres")

	// The cool, refreshing taste of Pepsi.
	refresher := &refresh.Service{
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

	updateCtx, updateCancel := context.WithCancel(ctx)
	defer func() {
		updateCancel()
	}()

	go refresher.Run(updateCtx)

	if err := s.Run(ctx); err != nil {
		err = fmt.Errorf("error running server: %w", err)
	}

	return err
}
