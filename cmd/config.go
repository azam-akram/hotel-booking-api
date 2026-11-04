package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const defaultPort = "9000"

type config struct {
	port        string
	databaseURL string
	logLevel    slog.Level
	enableAdmin bool
}

func loadConfig() (config, error) {
	cfg := config{
		port:        getenv("PORT", defaultPort),
		databaseURL: getenv("DATABASE_URL", ""),
	}
	if cfg.databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required, for example postgres://user:password@localhost:5432/dbname?sslmode=disable")
	}
	logLevel := getenv("LOG_LEVEL", "INFO")

	// parse log level text value and convert into slog.Level
	if err := cfg.logLevel.UnmarshalText([]byte(logLevel)); err != nil {
		cfg.logLevel = slog.LevelInfo
	}

	var err error
	if cfg.enableAdmin, err = strconv.ParseBool(getenv("ENABLE_ADMIN", "true")); err != nil {
		return config{}, fmt.Errorf("ENABLE_ADMIN: %w", err)
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
