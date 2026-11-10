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
	Content       ContentConfig
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

// ContentConfig is where entry bodies and images live: one private bucket, one KMS key.
type ContentConfig struct {
	Region    string
	Bucket    string
	KMSKeyARN string
}

var (
	// A user pool ID looks like "ap-southeast-2_AbC123xyZ": the region, an underscore, then an ID.
	userPoolIDPattern = regexp.MustCompile(`^([a-z]{2}(?:-[a-z]+)+-\d+)_[A-Za-z0-9]+$`)
	// S3 bucket names: 3-63 characters, lowercase letters, numbers, dots and hyphens.
	bucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
	// A full key ARN, e.g. arn:aws:kms:ap-southeast-2:123456789012:key/1234abcd-12ab-34cd-56ef-1234567890ab
	kmsKeyARNPattern = regexp.MustCompile(`^arn:aws:kms:[a-z0-9-]+:\d{12}:key/[0-9a-f-]{36}$`)
)

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

	bucket := strings.TrimSpace(os.Getenv("S3_CONTENT_BUCKET"))
	if !bucketPattern.MatchString(bucket) {
		return Config{}, fmt.Errorf("invalid S3_CONTENT_BUCKET %q: expected a bucket name like inkwell-dev-content-123456789012", bucket)
	}

	keyARN := strings.TrimSpace(os.Getenv("KMS_KEY_ARN"))
	if !kmsKeyARNPattern.MatchString(keyARN) {
		return Config{}, errors.New("invalid KMS_KEY_ARN: expected the full key ARN, arn:aws:kms:<region>:<account>:key/<key-id>")
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
		Content: ContentConfig{
			Region:    getEnv("AWS_REGION", match[1]), // default: the user pool's region
			Bucket:    bucket,
			KMSKeyARN: keyARN,
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
