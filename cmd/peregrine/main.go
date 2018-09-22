package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

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

	server := server.New(tba, c.Server.Address, year)
	if err := server.Run(); err != nil {
		panic(err)
	}
}
