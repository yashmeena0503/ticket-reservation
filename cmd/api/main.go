package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"reservation/internal/platform/config"
	"reservation/internal/platform/database"
	"time"
)

func main() {
	ctx := context.Background() // TODO: swap for graceful shutdown context

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

	// start listening on port
	slog.Info("listening", "port", cfg.Port)
	err = server.ListenAndServe()
	if err != nil {
		slog.Error("server error", "err", err)
	}
}

func health(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}
