package config

import (
	"os"
	"strconv"
	"strings"
)

// ServerConfig holds server-specific configuration values.
type ServerConfig struct {
	Port int
}

// AIConfig holds the LLM provider configuration values.
type AIConfig struct {
	Model   string
	BaseURL string
	APIKey  string
	Timeout string
}

// Config is the root configuration for the backend.
type Config struct {
	Server ServerConfig
	AI     AIConfig
}

// Load reads configuration from the environment and returns a Config object.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		AI: AIConfig{
			Model:   getenv("GEMINI_MODEL", "gemini-3.1-flash-lite-preview"),
			BaseURL: getenv("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"),
			APIKey:  strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
			Timeout: getenv("GEMINI_TIMEOUT", "2m"),
		},
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

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
