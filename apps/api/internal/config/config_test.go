package config

import (
	"log/slog"
	"testing"
)

const (
	testMongoURI   = "mongodb://localhost:27017"
	testUserPoolID = "ap-southeast-2_AbC123xyZ"
	testClientID   = "testclientid123"
)

// setValidEnv sets every required variable, so each test only changes what it is testing.
func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("MONGODB_URI", testMongoURI)
	t.Setenv("MONGODB_DATABASE", "")
	t.Setenv("COGNITO_USER_POOL_ID", testUserPoolID)
	t.Setenv("COGNITO_CLIENT_ID", testClientID)
}

func TestLoadDefaults(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
	if cfg.MongoURI != testMongoURI {
		t.Errorf("MongoURI = %q, want %q", cfg.MongoURI, testMongoURI)
	}
	if cfg.MongoDatabase != "inkwell" {
		t.Errorf("MongoDatabase = %q, want %q", cfg.MongoDatabase, "inkwell")
	}
}

func TestLoadOverrides(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("MONGODB_DATABASE", "inkwell_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
	if cfg.MongoDatabase != "inkwell_test" {
		t.Errorf("MongoDatabase = %q, want %q", cfg.MongoDatabase, "inkwell_test")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{name: "not a number", port: "abc"},
		{name: "zero", port: "0"},
		{name: "negative", port: "-1"},
		{name: "too large", port: "70000"},
		{name: "has spaces", port: " 8080"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("PORT", tc.port)

			if _, err := Load(); err == nil {
				t.Errorf("Load() with PORT=%q: expected an error, got nil", tc.port)
			}
		})
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	setValidEnv(t)
	t.Setenv("LOG_LEVEL", "loud")

	if _, err := Load(); err == nil {
		t.Error(`Load() with LOG_LEVEL="loud": expected an error, got nil`)
	}
}

func TestLoadRequiresMongoURI(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MONGODB_URI", "")

	if _, err := Load(); err == nil {
		t.Error("Load() without MONGODB_URI: expected an error, got nil")
	}
}

func TestLoadCognito(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	c := cfg.Cognito
	if c.Region != "ap-southeast-2" {
		t.Errorf("Region = %q, want %q", c.Region, "ap-southeast-2")
	}
	wantIssuer := "https://cognito-idp.ap-southeast-2.amazonaws.com/ap-southeast-2_AbC123xyZ"
	if got := c.Issuer(); got != wantIssuer {
		t.Errorf("Issuer() = %q, want %q", got, wantIssuer)
	}
	if got, want := c.JWKSURL(), wantIssuer+"/.well-known/jwks.json"; got != want {
		t.Errorf("JWKSURL() = %q, want %q", got, want)
	}
	if got, want := c.Endpoint(), "https://cognito-idp.ap-southeast-2.amazonaws.com/"; got != want {
		t.Errorf("Endpoint() = %q, want %q", got, want)
	}
}

func TestLoadRejectsInvalidCognito(t *testing.T) {
	tests := []struct {
		name     string
		poolID   string
		clientID string
	}{
		{name: "missing pool ID", poolID: "", clientID: testClientID},
		{name: "pool ID without region", poolID: "AbC123xyZ", clientID: testClientID},
		{name: "pool ID with bad characters", poolID: "ap-southeast-2_AbC-123", clientID: testClientID},
		{name: "missing client ID", poolID: testUserPoolID, clientID: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("COGNITO_USER_POOL_ID", tc.poolID)
			t.Setenv("COGNITO_CLIENT_ID", tc.clientID)

			if _, err := Load(); err == nil {
				t.Error("Load(): expected an error, got nil")
			}
		})
	}
}
