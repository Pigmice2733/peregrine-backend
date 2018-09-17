package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
)

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

	if err := http.ListenAndServe(c.Server.Address, nil); err != nil {
		panic(err)
	}
}
