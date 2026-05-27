package server

import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/jackc/pgx/v5/pgxpool"
)

// runMigrations applies any unapplied *.sql files from migrationsDir in
// alphabetical order, tracking applied files in a schema_migrations table.
func runMigrations(ctx context.Context, db *pgxpool.Pool, migrationsDir string) error {
    // Create tracking table if it doesn't exist.
    _, err := db.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            filename   TEXT PRIMARY KEY,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`)
    if err != nil {
        return fmt.Errorf("create schema_migrations: %w", err)
    }

    entries, err := os.ReadDir(migrationsDir)
    if err != nil {
        if os.IsNotExist(err) {
            return nil // no migrations directory is fine
        }
        return fmt.Errorf("read migrations dir: %w", err)
    }

    var filenames []string
    for _, e := range entries {
        if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
            filenames = append(filenames, e.Name())
        }
    }
    sort.Strings(filenames)

    for _, name := range filenames {
        var count int
        db.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE filename=$1`, name).Scan(&count)
        if count > 0 {
            continue
        }

        content, err := os.ReadFile(filepath.Join(migrationsDir, name))
        if err != nil {
            return fmt.Errorf("read %s: %w", name, err)
        }

        if _, err := db.Exec(ctx, string(content)); err != nil {
            return fmt.Errorf("apply %s: %w", name, err)
        }
        db.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, name)
        log.Printf("migration applied: %s", name)
    }
    return nil
}
