package store

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// newTestStore connects to TEST_DATABASE_URL and returns a Postgres bound to a
// private, empty schema that holds only the hotels table. The schema is dropped
// when the test ends, so existing data is never touched. The test is skipped
// when TEST_DATABASE_URL is not set.
func newTestStore(t *testing.T) *Postgres {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		// CI sets REQUIRE_DB so a lost variable fails the job instead of skipping.
		if os.Getenv("REQUIRE_DB") != "" {
			t.Fatal("TEST_DATABASE_URL not set but REQUIRE_DB is set")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect admin pool: %v", err)
	}
	t.Cleanup(admin.Close)

	schema := fmt.Sprintf("store_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `
		CREATE TABLE hotels (
			id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			name TEXT NOT NULL
		)`); err != nil {
		t.Fatalf("create hotels table: %v", err)
	}

	return New(pool)
}

func TestFindHotels(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Inserted in a non-alphabetical order to prove ORDER BY is applied.
	for _, name := range []string{
		"Seaside Inn",
		"Grand Palace Hotel",
		"Grand Palace Hotel Annex",
		"Ocean View Resort",
		"100% Cotton Lodge",
	} {
		if _, err := s.pool.Exec(ctx, `INSERT INTO hotels (name) VALUES ($1)`, name); err != nil {
			t.Fatalf("seed %q: %v", name, err)
		}
	}

	tests := []struct {
		name  string
		query string
		want  []string // expected hotel names, in order
	}{
		{
			name:  "exact match",
			query: "Ocean View Resort",
			want:  []string{"Ocean View Resort"},
		},
		{
			name:  "match ignores case",
			query: "oCeAn vIeW",
			want:  []string{"Ocean View Resort"},
		},
		{
			name:  "substring matches several hotels in name order",
			query: "grand",
			want:  []string{"Grand Palace Hotel", "Grand Palace Hotel Annex"},
		},
		{
			name:  "exact match is listed before longer names",
			query: "grand palace hotel",
			want:  []string{"Grand Palace Hotel", "Grand Palace Hotel Annex"},
		},
		{
			name:  "percent sign is literal, not a wildcard",
			query: "%",
			want:  []string{"100% Cotton Lodge"},
		},
		{
			name:  "underscore is literal, not a wildcard",
			query: "_",
			want:  []string{},
		},
		{
			name:  "no match returns an empty slice",
			query: "nowhere",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.FindHotels(ctx, tt.query)
			if err != nil {
				t.Fatalf("FindHotels(%q) error: %v", tt.query, err)
			}
			if got == nil {
				t.Fatalf("FindHotels(%q) returned nil, want a non-nil slice", tt.query)
			}

			names := make([]string, 0, len(got))
			for _, h := range got {
				if h.ID == 0 {
					t.Errorf("hotel %q has zero ID", h.Name)
				}
				names = append(names, h.Name)
			}
			if !reflect.DeepEqual(names, tt.want) {
				t.Errorf("FindHotels(%q) = %v, want %v", tt.query, names, tt.want)
			}
		})
	}
}

func TestFindHotelsReturnsErrorOnClosedPool(t *testing.T) {
	s := newTestStore(t)
	s.pool.Close()

	if _, err := s.FindHotels(context.Background(), "grand"); err == nil {
		t.Fatal("expected an error from a closed pool, got nil")
	}
}
