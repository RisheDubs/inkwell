package config

import (
	"log/slog"
	"testing"
)

const testMongoURI = "mongodb://localhost:27017"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("MONGODB_URI", testMongoURI)

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
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("MONGODB_URI", testMongoURI)

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
			t.Setenv("PORT", tc.port)
			t.Setenv("LOG_LEVEL", "")

			if _, err := Load(); err == nil {
				t.Errorf("Load() with PORT=%q: expected an error, got nil", tc.port)
			}
		})
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "loud")
	t.Setenv("MONGODB_URI", testMongoURI)

	if _, err := Load(); err == nil {
		t.Error(`Load() with LOG_LEVEL="loud": expected an error, got nil`)
	}
}

func TestLoadRequiresMongoURI(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("MONGODB_URI", "")

	if _, err := Load(); err == nil {
		t.Error("Load() without MONGODB_URI: expected an error, got nil")
	}
}
