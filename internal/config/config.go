package config

// Config holds information about how the peregrine backend is configured.
type Config struct {
	Server struct {
		Listen string `yaml:"listen"`
		Origin string `yaml:"origin"`
	} `yaml:"server"`
}
