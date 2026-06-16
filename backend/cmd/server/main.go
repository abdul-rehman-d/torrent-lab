package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"torrent-panel/backend/internal/config"
	"torrent-panel/backend/internal/httpapi"
	"torrent-panel/backend/internal/media"
	"torrent-panel/backend/internal/qbit"
	"torrent-panel/backend/internal/torrents"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	qbitClient := qbit.NewClient(cfg.QbitURL, cfg.QbitUsername, cfg.QbitPassword, cfg.DownloadsPath)
	sorter := media.NewSorter(cfg.DownloadsPath, cfg.MoviesPath, cfg.TVPath)
	service := torrents.NewService(qbitClient, sorter, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go service.StartCompletionWorker(ctx, 15*time.Second)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(service, qbitClient, cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve api", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown api", "error", err)
	}
}
