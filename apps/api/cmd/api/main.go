package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RisheDubs/inkwell/apps/api/internal/config"
	"github.com/RisheDubs/inkwell/apps/api/internal/httpapi"
	mongostore "github.com/RisheDubs/inkwell/apps/api/internal/store/mongo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("load config", "err", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	// ctx is cancelled when the process receives Ctrl+C (SIGINT) or a stop request (SIGTERM).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	store, err := mongostore.Connect(connectCtx, cfg.MongoURI)
	cancel()
	if err != nil {
		slog.Error("connect to mongodb", "err", err)
		os.Exit(1)
	}
	slog.Info("connected to mongodb")

	// Open the port first, so "listening" is only logged once it is true.
	ln, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		slog.Error("listen", "addr", ":"+cfg.Port, "err", err)
		_ = store.Close(context.Background())
		os.Exit(1)
	}

	srv := &http.Server{
		Handler:           httpapi.NewRouter(store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ln)
	}()
	slog.Info("api listening", "addr", ln.Addr().String(), "log_level", cfg.LogLevel.String())

	// Wait for either a server failure or a shutdown signal.
	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "err", err)
			_ = store.Close(context.Background())
			os.Exit(1)
		}
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	// Give in-flight requests up to 10 seconds to finish, then close MongoDB.
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown", "err", err)
	}
	if err := store.Close(shutdownCtx); err != nil {
		slog.Error("mongodb disconnect", "err", err)
	}
	slog.Info("api stopped cleanly")
}
