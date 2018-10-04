package config

import (
	"fmt"
	"os"
	"path"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	yaml "gopkg.in/yaml.v2"
)

// Config holds information about how the peregrine backend is configured.
type Config struct {
	Server struct {
		HTTPAddress  string `yaml:"httpAddress"`
		HTTPSAddress string `yaml:"httpsAddress"`
		KeyFile      string `yaml:"keyFile"`
		CertFile     string `yaml:"certFile"`
		Origin       string `yaml:"origin"`
		Year         int    `yaml:"year"`
	} `yaml:"server"`

	TBA struct {
		URL    string `yaml:"URL"`
		APIKey string `yaml:"apiKey"`
	} `yaml:"tba"`

	SeedUser *struct {
		Username  string      `yaml:"username"`
		Password  string      `yaml:"password"`
		FirstName string      `yaml:"firstName"`
		LastName  string      `yaml:"lastName"`
		Roles     store.Roles `yaml:"roles"`
	} `yaml:"seedUser"`

	Database store.Options `yaml:"database"`
}

// Open opens basePath/etc/config.$GO_ENV.json as a Config
func Open(basePath string) (Config, error) {
	environment := "development"
	if goEnv, ok := os.LookupEnv("GO_ENV"); ok {
		environment = goEnv
	}

	f, err := os.Open(path.Join(basePath, "etc", fmt.Sprintf("config.%s.yaml", environment)))
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	var c Config
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		return c, err
	}

	if apiKey, ok := os.LookupEnv("TBA_API_KEY"); ok {
		c.TBA.APIKey = apiKey
	}

	return c, nil
}
