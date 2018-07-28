package main

import (
	"os"
	"path/filepath"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/Pigmice2733/peregrine-backend/internal/server"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	logger := logrus.New()

	conf := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "Pigmice2733", "peregrine-backend", "etc", "config.yaml")
	if len(os.Args) >= 2 {
		conf = os.Args[1]
	}
	logger.Infof("using config file: %s", conf)

	f, err := os.Open(conf)
	if err != nil {
		logger.Fatalf("error opening config file: %v\n", err)
	}
	defer f.Close()

	var c config.Config
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		logger.Fatalf("error decoding config file: %v\n", err)
	}

	s := server.New(c, logger)

	logger.Infof("starting server listening on: %s", c.Server.Address)
	if err := s.Run(); err != nil {
		logger.Fatalf("error running server: %v", err)
	}
}
