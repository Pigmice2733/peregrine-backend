package config

// Config holds information about how the peregrine backend is configured.
type Config struct {
	Server struct {
		Address string `yaml:"address"`
		Origin  string `yaml:"origin"`
	} `yaml:"server"`
}
