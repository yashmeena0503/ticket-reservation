package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reservation/internal/auth"
	"reservation/internal/db"
	"reservation/internal/events"
	"reservation/internal/platform/config"
	"reservation/internal/platform/database"
	"reservation/internal/platform/httpx"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// init logger
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

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

	authService := auth.NewService(db.New(pool), cfg.JWTSecret)
	// init serve mux
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /ready", ready(pool))
	// endpoints with authorization
	auth.Register(mux, authService)
	events.Register(mux, pool, authService)
	// middlewares
	handler := httpx.Recovery(mux)
	handler = httpx.Logging(handler)
	handler = httpx.RequestId(handler)

	// init server
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
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

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func ready(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		err := pool.Ping(ctx)
		if err != nil {
			slog.Error("ready check failed", "err", err)
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
