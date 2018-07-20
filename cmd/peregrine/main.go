package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/Pigmice2733/peregrine-backend/internal/server"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	conf := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "Pigmice2733", "peregrine-backend", "etc", "config.yaml")
	if len(os.Args) >= 2 {
		conf = os.Args[1]
	}

	f, err := os.Open(conf)
	if err != nil {
		log.Fatalf("error opening config file: %v\n", err)
	}
	defer f.Close()

	var c config.Config
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		log.Fatalf("error decoding config file: %v\n", err)
	}

	s := server.New(c)

	if err := s.Run(); err != nil {
		log.Fatalln(err)
	}
}
