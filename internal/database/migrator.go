package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migrator applies versioned SQL migrations from the repository.
//
// Directory layout:
//
//	sql/mysql/*.sql
//	sql/postgres/*.sql
//
// Migrations are applied in filename order. Each file name (without extension)
// is used as the migration version recorded in schema_migrations.
type Migrator struct {
	baseDir string
}

func NewMigrator(baseDir string) *Migrator {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = "sql"
	}
	return &Migrator{baseDir: baseDir}
}

func (m *Migrator) Migrate(ctx context.Context, db *DB) error {
	if db == nil {
		return fmt.Errorf("db not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}

	applied, err := loadAppliedVersions(ctx, db)
	if err != nil {
		return err
	}

	dir := filepath.Join(m.baseDir, db.Dialect().Name())
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("migrations directory not found: %s", dir)
		}
		return fmt.Errorf("read migrations directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, filepath.Ext(name))
		if version == "" || applied[version] {
			continue
		}

		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", path, err)
		}

		statements := splitSQLStatements(string(content))
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				// Keep migrations restartable across MySQL/PG by tolerating common
				// already-exists errors (e.g. upgrading from legacy ensureSchema).
				if db.Dialect().IsDuplicateColumn(err) || db.Dialect().IsDuplicateIndex(err) {
					continue
				}
				return fmt.Errorf("migration %s failed: %w", name, err)
			}
		}

		if err := recordAppliedVersion(ctx, db, version); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		applied[version] = true
	}
	return nil
}

func ensureMigrationsTable(ctx context.Context, db Conn) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func loadAppliedVersions(ctx context.Context, db Conn) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		v = strings.TrimSpace(v)
		if v != "" {
			applied[v] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}
	return applied, nil
}

func recordAppliedVersion(ctx context.Context, db Conn, version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("empty migration version")
	}
	_, err := db.ExecContext(ctx,
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
		version,
		time.Now().UTC(),
	)
	if err != nil {
		// If a previous attempt applied the migration but failed to record it, we want
		// a restart to be safe. We therefore treat duplicate key as already applied.
		if db.Dialect().IsDuplicateKey(err) {
			return nil
		}
		return err
	}
	return nil
}

func splitSQLStatements(sqlText string) []string {
	var out []string
	var b strings.Builder

	inSingle := false
	inDouble := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(sqlText); i++ {
		ch := sqlText[i]
		var next byte
		if i+1 < len(sqlText) {
			next = sqlText[i+1]
		}

		if inLineComment {
			b.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			b.WriteByte(ch)
			if ch == '*' && next == '/' {
				b.WriteByte(next)
				i++
				inBlockComment = false
			}
			continue
		}

		// Start of comment (only when not in any string/identifier literal).
		if !inSingle && !inDouble && !inBacktick {
			if ch == '-' && next == '-' {
				inLineComment = true
				b.WriteByte(ch)
				b.WriteByte(next)
				i++
				continue
			}
			if ch == '/' && next == '*' {
				inBlockComment = true
				b.WriteByte(ch)
				b.WriteByte(next)
				i++
				continue
			}
		}

		if ch == '\'' && !inDouble && !inBacktick {
			b.WriteByte(ch)
			if inSingle {
				// SQL escapes a single quote by doubling it.
				if next == '\'' {
					b.WriteByte(next)
					i++
					continue
				}
				inSingle = false
			} else {
				inSingle = true
			}
			continue
		}
		if ch == '"' && !inSingle && !inBacktick {
			b.WriteByte(ch)
			if inDouble {
				if next == '"' {
					b.WriteByte(next)
					i++
					continue
				}
				inDouble = false
			} else {
				inDouble = true
			}
			continue
		}
		if ch == '`' && !inSingle && !inDouble {
			b.WriteByte(ch)
			inBacktick = !inBacktick
			continue
		}

		if ch == ';' && !inSingle && !inDouble && !inBacktick {
			stmt := strings.TrimSpace(b.String())
			if stmt != "" {
				out = append(out, stmt)
			}
			b.Reset()
			continue
		}

		b.WriteByte(ch)
	}

	if tail := strings.TrimSpace(b.String()); tail != "" {
		out = append(out, tail)
	}
	return out
}
