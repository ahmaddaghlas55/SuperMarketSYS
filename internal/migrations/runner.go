package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Files are embedded so the compiled binary can migrate a fresh Termux database
// without depending on files beside the executable.
//
//go:embed *.sql
var files embed.FS

type migration struct {
	version int
	name    string
}

func Run(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	entries, err := files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	var pending []migration
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		version, parseErr := strconv.Atoi(strings.SplitN(entry.Name(), "_", 2)[0])
		if parseErr != nil {
			return fmt.Errorf("invalid migration filename %q: %w", entry.Name(), parseErr)
		}
		pending = append(pending, migration{version: version, name: entry.Name()})
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].version < pending[j].version })

	for _, migration := range pending {
		var applied int
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, migration.version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", migration.version, err)
		}
		if applied != 0 {
			continue
		}
		content, err := files.ReadFile(migration.name)
		if err != nil {
			return fmt.Errorf("read migration %d: %w", migration.version, err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", migration.version, err)
		}
		if _, err = tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", migration.version, err)
		}
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO schema_migrations(version) VALUES (?)`, migration.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", migration.version, err)
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", migration.version, err)
		}
	}
	return nil
}
