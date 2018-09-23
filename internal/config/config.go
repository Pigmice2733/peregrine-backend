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
		Address string `yaml:"address"`
		Origin  string `yaml:"origin"`
	} `yaml:"server"`

	TBA struct {
		URL    string `yaml:"URL"`
		APIKey string `yaml:"apiKey"`
	} `yaml:"tba"`

	Database store.Options `yaml:"database"`
}

// Open opens basePath/etc/config.{environment}.json as a Config
func Open(basePath string, environment string) (Config, error) {
	f, err := os.Open(path.Join(basePath, "etc", fmt.Sprintf("config.%s.yaml", environment)))
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	var c Config
	err = yaml.NewDecoder(f).Decode(&c)
	return c, err
}
