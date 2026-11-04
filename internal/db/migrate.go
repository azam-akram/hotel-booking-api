package db

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// sqlFS holds the schema migrations and the seed/reset scripts, built into the binary.
//
//go:embed migrations/*.sql seeds/*.sql
var sqlFS embed.FS

// Seed returns the SQL that inserts the test data.
func Seed() string {
	return mustRead("seeds/seed.sql")
}

// Reset returns the SQL that removes all data.
func Reset() string {
	return mustRead("seeds/reset.sql")
}

func mustRead(name string) string {
	b, err := sqlFS.ReadFile(name)
	if err != nil {
		panic(err) // the files are embedded at build time, so this cannot happen
	}
	return string(b)
}

// Migrate applies all pending up migrations. It is a no-op when the schema is current.
func Migrate(databaseURL string) error {
	src, err := iofs.New(sqlFS, "migrations")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	// the pgx/v5 driver registers under the "pgx5" scheme
	url := strings.Replace(databaseURL, "postgres://", "pgx5://", 1)
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
