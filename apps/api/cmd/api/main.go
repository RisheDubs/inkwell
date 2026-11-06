package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/RisheDubs/inkwell/apps/api/internal/config"
	"github.com/RisheDubs/inkwell/apps/api/internal/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("load config", "err", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("api starting", "addr", srv.Addr, "log_level", cfg.LogLevel.String())
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
