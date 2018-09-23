package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

// Config holds information about how the peregrine backend is configured.
type Config struct {
	Server struct {
		Address string `json:"address"`
		Origin  string `json:"origin"`
	} `json:"server"`

	TBA struct {
		URL    string `json:"URL"`
		APIKey string `json:"apiKey"`
	} `json:"tba"`

	Database struct {
		User    string `json:"user"`
		Pass    string `json:"password"`
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Name    string `json:"name"`
		SSLMode string `json:"sslMode"`
	} `json:"database"`
}

// Open opens basePath/etc/config.{environment}.json as a Config
func Open(basePath string, environment string) (Config, error) {
	f, err := os.Open(path.Join(basePath, "etc", fmt.Sprintf("config.%s.json", environment)))
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	var c Config
	err = json.NewDecoder(f).Decode(&c)
	return c, err
}
