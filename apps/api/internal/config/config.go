// Package config loads API settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Config holds every setting the API needs at startup.
type Config struct {
	Port     string
	LogLevel slog.Level
	MongoURI string
}

// Load reads configuration from the environment, applies defaults,
// and returns an error if any value is invalid.
func Load() (Config, error) {
	port := getEnv("PORT", "8080")
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("invalid PORT %q: must be a number from 1 to 65535", port)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(getEnv("LOG_LEVEL", "info"))); err != nil {
		return Config{}, fmt.Errorf("invalid LOG_LEVEL: %w", err)
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		return Config{}, errors.New("MONGODB_URI is required")
	}

	return Config{Port: port, LogLevel: level, MongoURI: mongoURI}, nil

}

// getEnv returns the environment variable's value, or fallback if it is unset or empty.
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
