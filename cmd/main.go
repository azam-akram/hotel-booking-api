package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/azam-akram/hotel-booking-api/internal/db"
	"github.com/azam-akram/hotel-booking-api/internal/handler"
	"github.com/azam-akram/hotel-booking-api/internal/service"
	"github.com/azam-akram/hotel-booking-api/internal/store"
)

const (
	shutdownTimeout = 10 * time.Second
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid configuration:", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.logLevel,
	}))
	slog.SetDefault(logger)

	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(cfg config, log *slog.Logger) error {
	// Stop on Ctrl+C or SIGTERM (docker stop).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("starting", "port", cfg.port, "log_level", cfg.logLevel.String())

	log.Info("applying database migrations")
	if err := db.Migrate(cfg.databaseURL); err != nil {
		return err
	}
	log.Info("database migrations up to date")

	pool, err := openDatabase(ctx, cfg.databaseURL, log)
	if err != nil {
		return err
	}
	defer pool.Close()

	postgres := store.New(pool)
	svc := service.New(postgres, log)

	h := handler.New(svc, log)
	server := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           handler.NewRouter(h, log, cfg.enableAdmin),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server starting", "addr", server.Addr, "admin_enabled", cfg.enableAdmin)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	// Give the server 10 seconds to finish active requests during shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err = server.Shutdown(shutdownCtx)

	if err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Info("server stopped")
	return nil
}
