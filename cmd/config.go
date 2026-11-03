package main

import (
	"log/slog"
	"os"
	"strings"
)

const defaultPort = "9000"

type config struct {
	port        string
	databaseURL string
	logLevel    slog.Level
}

const defaultDatabaseURL = "postgres://roomuser:password1@localhost:5433/roomdb?sslmode=disable"

func loadConfig() config {
	cfg := config{
		port:        getenv("PORT", defaultPort),
		databaseURL: getenv("DATABASE_URL", defaultDatabaseURL),
	}
	logLevel := getenv("LOG_LEVEL", "INFO")

	// parse log level text value and convert into slog.Level
	if err := cfg.logLevel.UnmarshalText([]byte(logLevel)); err != nil {
		cfg.logLevel = slog.LevelInfo
	}

	return cfg
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
