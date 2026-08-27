// Package db manages the Postgres connection pool and startup migrations.
package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &DB{Pool: pool}, nil
}

func (d *DB) Close() {
	d.Pool.Close()
}

// RunMigrations applies every .up.sql file found in dir, in filename order,
// tracking applied migrations in a schema_migrations table.
func (d *DB) RunMigrations(ctx context.Context, dir string) error {
	if _, err := d.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := readSQLFiles(dir)
	if err != nil {
		return err
	}
	sort.Strings(entries)

	for _, path := range entries {
		if !strings.HasSuffix(path, ".up.sql") {
			continue
		}
		version := strings.TrimSuffix(pathBase(path), ".up.sql")

		var applied bool
		err := d.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}

		content, err := readFile(path)
		if err != nil {
			return err
		}

		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		for _, stmt := range splitStatements(content) {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := tx.Exec(ctx, stmt); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("apply migration %s: %w", version, err)
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func splitStatements(sql string) []string {
	return strings.Split(sql, ";\n")
}

func readSQLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sql") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	return paths, nil
}

func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read migration file %s: %w", path, err)
	}
	return string(content), nil
}

func pathBase(path string) string {
	return filepath.Base(path)
}
