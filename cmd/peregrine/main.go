package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/Pigmice2733/peregrine-backend/internal/server"
)

var year = time.Now().Year()

func main() {
	basePath := "."
	if len(os.Args) > 1 {
		basePath = os.Args[2]
	}

	env := "development"
	if e, ok := os.LookupEnv("GO_ENV"); ok {
		env = e
	}

	f, err := os.Open(path.Join(basePath, "etc", fmt.Sprintf("config.%s.json", env)))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var c config.Config
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		panic(err)
	}

	tbaKey, ok := os.LookupEnv("TBA_API_KEY")
	if !ok {
		fmt.Println("Error: TBA_API_KEY env var not set")
		os.Exit(0)
	}

	tba := tba.Service{
		URL:    c.TBA.URL,
		APIKey: tbaKey,
	}

	port, err := strconv.Atoi(os.Getenv("PG_PORT"))
	if err != nil {
		port = 5432
		fmt.Printf("PG_PORT defaulted to: %d\n", port)
	}

	store, err := store.New(store.Options{
		User:    os.Getenv("PG_USER"),
		Pass:    os.Getenv("PG_PASS"),
		Host:    os.Getenv("PG_HOST"),
		Port:    port,
		DBName:  os.Getenv("PG_DB_NAME"),
		SSLMode: os.Getenv("PG_SSL_MODE"),
	})
	if err != nil {
		fmt.Printf("unable to connect to postgres server: %v\n", err)
		os.Exit(1)
	}

	server := server.New(tba, store, c.Server.Address, year)
	if err := server.Run(); err != nil {
		panic(err)
	}
}
