package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/Pigmice2733/peregrine-backend/internal/server"
)

var year = time.Now().Year()

func main() {
	var basePath = flag.String("basePath", ".", "Path to the etc directory where the config file is.")

	flag.Parse()

	env := "development"
	if e, ok := os.LookupEnv("GO_ENV"); ok {
		env = e
	}

	c, err := config.Open(*basePath, env)
	if err != nil {
		fmt.Printf("Error: opening config: %v\n", err)
		return
	}

	tba := tba.Service{
		URL:    c.TBA.URL,
		APIKey: c.TBA.APIKey,
	}

	store, err := store.New(c.Database)
	if err != nil {
		fmt.Printf("Error: unable to connect to postgres server: %v\n", err)
		return
	}

	server := server.New(tba, store, c.Server.Address, c.Server.Origin, year)

	fmt.Printf("Starting server at: %s\n", c.Server.Address)
	if err := server.Run(); err != nil {
		fmt.Printf("Error: server.Run: %v\n", err)
	}
}
