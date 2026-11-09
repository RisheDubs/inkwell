// Package config loads API settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Config holds every setting the API needs at startup.
type Config struct {
	Port          string
	LogLevel      slog.Level
	MongoURI      string
	MongoDatabase string
	Cognito       CognitoConfig
}

// CognitoConfig identifies the Cognito user pool and app client whose tokens the API accepts.
type CognitoConfig struct {
	Region     string
	UserPoolID string
	ClientID   string
}

// Issuer is the value Cognito puts in every token's "iss" claim for this user pool.
func (c CognitoConfig) Issuer() string {
	return fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", c.Region, c.UserPoolID)
}

// JWKSURL is where Cognito publishes the public keys that verify this pool's tokens.
func (c CognitoConfig) JWKSURL() string {
	return c.Issuer() + "/.well-known/jwks.json"
}

// Endpoint is the Cognito API endpoint for the pool's region.
func (c CognitoConfig) Endpoint() string {
	return fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/", c.Region)
}

// A user pool ID looks like "ap-southeast-2_AbC123xyZ": the region, an underscore, then an ID.
var userPoolIDPattern = regexp.MustCompile(`^([a-z]{2}(?:-[a-z]+)+-\d+)_[A-Za-z0-9]+$`)

// Load reads configuration from the environment, applies defaults,
// and returns an error if any value is missing or invalid.
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

	poolID := strings.TrimSpace(os.Getenv("COGNITO_USER_POOL_ID"))
	match := userPoolIDPattern.FindStringSubmatch(poolID)
	if match == nil {
		return Config{}, fmt.Errorf("invalid COGNITO_USER_POOL_ID %q: expected something like ap-southeast-2_AbC123xyZ", poolID)
	}

	clientID := strings.TrimSpace(os.Getenv("COGNITO_CLIENT_ID"))
	if clientID == "" {
		return Config{}, errors.New("COGNITO_CLIENT_ID is required")
	}

	return Config{
		Port:          port,
		LogLevel:      level,
		MongoURI:      mongoURI,
		MongoDatabase: getEnv("MONGODB_DATABASE", "inkwell"),
		Cognito: CognitoConfig{
			Region:     match[1],
			UserPoolID: poolID,
			ClientID:   clientID,
		},
	}, nil
}

// getEnv returns the environment variable's value, or fallback if it is unset or empty.
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
