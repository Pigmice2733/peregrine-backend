package config

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v9"
)

// Server holds information about the peregrine backend HTTP server.
type Server struct {
	Listen    string       `json:"listen" validate:"required"`
	Origin    string       `json:"origin" validate:"required"`
	LogLevel  logrus.Level `json:"logLevel"`
	LogJSON   bool         `json:"logJSON"`
	JWTSecret string       `json:"jwtSecret" validate:"required,min=32"`
}

// Config holds information about how the peregrine backend is configured.
type Config struct {
	Server Server `json:"server" validate:"dive"`
	Year   int    `json:"year" validate:"required"`
	TBA    struct {
		URL    string `validate:"required"`
		APIKey string `validate:"required"`
	} `json:"tba"`
	DSN string `json:"dsn" validate:"required"`
}

// Open parses and validates the JSON config at the given path.
func Open(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, errors.Wrap(err, "unable to open file")
	}
	defer f.Close()

	var c Config
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return Config{}, errors.Wrap(err, "unable to unmarshal file")
	}

	validate := validator.New()
	return c, errors.Wrap(validate.Struct(c), "config loaded from environment variables fails to validate")
}
