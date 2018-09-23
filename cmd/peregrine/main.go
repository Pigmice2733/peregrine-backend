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

	store, err := store.New(store.Options{
		User:    c.Database.User,
		Pass:    c.Database.Pass,
		Host:    c.Database.Host,
		Port:    c.Database.Port,
		DBName:  c.Database.Name,
		SSLMode: c.Database.SSLMode,
	})
	if err != nil {
		fmt.Printf("Error: unable to connect to postgres server: %v\n", err)
		return
	}

	server := server.New(tba, store, c.Server.Address, c.Server.Origin, year)
	if err := server.Run(); err != nil {
		fmt.Printf("Error: server.Run: %v\n", err)
	}
}
