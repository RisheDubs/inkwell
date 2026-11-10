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

	awsconfig "github.com/aws/aws-sdk-go-v2/config" // 3B
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3" // 3B

	"github.com/RisheDubs/inkwell/apps/api/internal/auth"
	"github.com/RisheDubs/inkwell/apps/api/internal/config"
	"github.com/RisheDubs/inkwell/apps/api/internal/httpapi"
	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
	mongostore "github.com/RisheDubs/inkwell/apps/api/internal/store/mongo"
	s3store "github.com/RisheDubs/inkwell/apps/api/internal/store/s3" // 3B
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
	store, err := mongostore.Connect(connectCtx, cfg.MongoURI, cfg.MongoDatabase)
	if err == nil {
		err = store.EnsureIndexes(connectCtx)
	}
	cancel()
	if err != nil {
		slog.Error("set up mongodb", "err", err)
		os.Exit(1)
	}
	slog.Info("connected to mongodb", "database", cfg.MongoDatabase)

	// Downloads Cognito's public keys and refreshes them in the background until ctx ends.
	verifier, err := auth.NewCognitoVerifier(ctx, cfg.Cognito.JWKSURL(), cfg.Cognito.Issuer(), cfg.Cognito.ClientID)
	if err != nil {
		slog.Error("set up token verifier", "err", err)
		_ = store.Close(context.Background())
		os.Exit(1)
	}

	svc := journal.NewService(store, auth.NewUserInfoClient(cfg.Cognito.Endpoint()))

	// 3B: AWS credentials come from the environment: one-hour role credentials locally
	// (make creds), the ECS task role in production. The code is the same either way.
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Content.Region))
	if err != nil {
		slog.Error("load aws config", "err", err)
		_ = store.Close(context.Background())
		os.Exit(1)
	}
	bodies := s3store.NewBodies(awss3.NewFromConfig(awsCfg), cfg.Content.Bucket, cfg.Content.KMSKeyARN)
	entries := journal.NewEntryService(store, store, bodies, mongostore.NewID)

	// Open the port first, so "listening" is only logged once it is true.
	ln, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		slog.Error("listen", "addr", ":"+cfg.Port, "err", err)
		_ = store.Close(context.Background())
		os.Exit(1)
	}

	srv := &http.Server{
		Handler:           httpapi.NewRouter(httpapi.Deps{DB: store, Verifier: verifier, Me: svc, Journals: svc, Entries: entries}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ln)
	}()
	slog.Info("api listening",
		"addr", ln.Addr().String(),
		"log_level", cfg.LogLevel.String(),
		"user_pool", cfg.Cognito.UserPoolID,
		"bucket", cfg.Content.Bucket, // 3B
	)

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
