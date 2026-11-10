package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reservation/internal/platform/config"
	"reservation/internal/platform/database"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// init config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("error loading config", "err", err)
		os.Exit(1)
	}

	// connect to postgres
	pool, err := database.Connect(ctx, cfg.DatabaseUrl)
	if err != nil {
		slog.Error("error connecting to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// init serve mux
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)

	// init server
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "port", cfg.Port)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		// server died by itself
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
		}
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			slog.Error("shutdown error", "err", err)
		}
	}
}

func health(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}
