package config

import (
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Server holds information about the peregrine backend HTTP server.
type Server struct {
	HTTPAddress  string `yaml:"httpAddress"`
	HTTPSAddress string `yaml:"httpsAddress"`
	KeyFile      string `yaml:"keyFile"`
	CertFile     string `yaml:"certFile"`
	Origin       string `yaml:"origin"`
	Year         int    `yaml:"year"`
	LogJSON      bool   `yaml:"logJSON"`
	Secret       string `yaml:"secret"`
}

// Config holds information about how the peregrine backend is configured.
type Config struct {
	Server Server `yaml:"server"`

	TBA struct {
		URL    string `yaml:"URL"`
		APIKey string `yaml:"apiKey"`
	} `yaml:"tba"`

	DSN string `yaml:"dsn"`
}

// Open opens basePath/etc/config.$GO_ENV.json as a Config
func Open(basePath string) (Config, error) {
	environment := "development"
	if goEnv, ok := os.LookupEnv("GO_ENV"); ok {
		environment = goEnv
	}

	viper.AddConfigPath(path.Join(basePath, "etc"))
	viper.SetConfigName("config." + environment)

	secretUUID, err := uuid.NewRandom()
	if err != nil {
		return Config{}, errors.Wrap(err, "unable to generate uuid for secret")
	}

	viper.SetDefault("server", map[string]interface{}{
		"httpAddress": ":8080",
		"origin":      "*",
		"year":        time.Now().Year(),
		"logJSON":     false,
		"secret":      secretUUID.String(),
	})
	viper.SetDefault("tba.url", "https://www.thebluealliance.com/api/v3")
	if err := viper.BindEnv("tba.apiKey", "PEREGRINE_TBA_API_KEY"); err != nil {
		return Config{}, errors.Wrap(err, "unable to bind viper env var for api key")
	}

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, errors.Wrap(err, "unable to read in config file")
	}

	var c Config
	if err := viper.Unmarshal(&c); err != nil {
		return c, errors.Wrap(err, "unable to unmarshal config from viper")
	}

	return c, nil
}
