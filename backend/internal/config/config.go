package config

import (
	"os"
	"strconv"
)

// ServerConfig holds server-specific configuration values.
type ServerConfig struct {
	Port int
}

// Config is the root configuration for the backend.
type Config struct {
	Server ServerConfig
}

// Load reads configuration from the environment and returns a Config object.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
	}

	if portStr := os.Getenv("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, err
		}
		cfg.Server.Port = port
	}

	return cfg, nil
}
