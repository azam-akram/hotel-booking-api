package main

import (
	"log/slog"
	"os"
)

const defaultPort = "9000"

type config struct {
	port     string
	logLevel slog.Level
}

func loadConfig() config {
	cfg := config{
		port: getenv("PORT", defaultPort),
	}
	logLevel := getenv("LOG_LEVEL", "INFO")

	// parse log level text value and convert into slog.Level
	if err := cfg.logLevel.UnmarshalText([]byte(logLevel)); err != nil {
		cfg.logLevel = slog.LevelInfo
	}

	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
