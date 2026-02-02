package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// InsertReturningID executes an INSERT statement and returns the auto-generated id.
//
// Callers should provide an INSERT statement without a trailing semicolon.
// For PostgreSQL, the function appends "RETURNING id" and uses QueryRowContext.
// For MySQL, it uses ExecContext + LastInsertId.
func InsertReturningID(ctx context.Context, db Conn, query string, args ...any) (int64, error) {
	query = strings.TrimSpace(query)
	query = strings.TrimSuffix(query, ";")

	switch db.Dialect().Name() {
	case "postgres":
		q := query + " RETURNING id"
		var id int64
		if err := db.QueryRowContext(ctx, q, args...).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	default:
		res, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
}

// ExecInsertIgnore executes a best-effort "insert if not exists".
//
// MySQL uses "INSERT IGNORE".
// PostgreSQL uses "ON CONFLICT (...) DO NOTHING" and therefore requires conflictCols.
func ExecInsertIgnore(ctx context.Context, db Conn, table string, cols []string, conflictCols []string, args ...any) (sql.Result, error) {
	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(table)
	b.WriteString(" (")
	b.WriteString(strings.Join(cols, ", "))
	b.WriteString(") VALUES (")
	b.WriteString(placeholders(len(cols)))
	b.WriteString(")")

	switch db.Dialect().Name() {
	case "postgres":
		b.WriteString(" ON CONFLICT (")
		b.WriteString(strings.Join(conflictCols, ", "))
		b.WriteString(") DO NOTHING")
	default:
		// MySQL
		// Replace "INSERT INTO" with "INSERT IGNORE INTO".
		// Keep it simple to avoid a full SQL builder.
		sqlText := b.String()
		sqlText = strings.Replace(sqlText, "INSERT INTO", "INSERT IGNORE INTO", 1)
		return db.ExecContext(ctx, sqlText, args...)
	}

	return db.ExecContext(ctx, b.String(), args...)
}

// ExecUpsert executes an INSERT ... ON DUPLICATE KEY/ON CONFLICT DO UPDATE.
//
// updateCols are always overwritten by the incoming value.
// updateCoalesceCols are only overwritten when the incoming value is non-NULL (best-effort to
// mirror legacy MySQL collation/behavior where callers pass NULL to mean "keep existing").
//
// For PostgreSQL, conflictCols is required (it becomes the ON CONFLICT (...) target).
func ExecUpsert(
	ctx context.Context,
	db Conn,
	table string,
	cols []string,
	conflictCols []string,
	updateCols []string,
	updateCoalesceCols []string,
	args ...any,
) (sql.Result, error) {
	if db == nil {
		return nil, fmt.Errorf("db not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("empty insert cols")
	}
	if len(args) != len(cols) {
		return nil, fmt.Errorf("args mismatch: got %d, want %d", len(args), len(cols))
	}

	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(table)
	b.WriteString(" (")
	b.WriteString(strings.Join(cols, ", "))
	b.WriteString(") VALUES (")
	b.WriteString(placeholders(len(cols)))
	b.WriteString(")")

	// If there is no update clause, treat it as "insert ignore".
	if len(updateCols) == 0 && len(updateCoalesceCols) == 0 {
		return ExecInsertIgnore(ctx, db, table, cols, conflictCols, args...)
	}

	switch db.Dialect().Name() {
	case "postgres":
		if len(conflictCols) == 0 {
			return nil, fmt.Errorf("postgres upsert requires conflictCols")
		}

		b.WriteString(" ON CONFLICT (")
		b.WriteString(strings.Join(conflictCols, ", "))
		b.WriteString(") DO UPDATE SET ")

		assignments := make([]string, 0, len(updateCols)+len(updateCoalesceCols))
		for _, c := range updateCols {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			assignments = append(assignments, c+" = EXCLUDED."+c)
		}
		for _, c := range updateCoalesceCols {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			// In Postgres, we can reference the existing row directly by table name.
			assignments = append(assignments, c+" = COALESCE(EXCLUDED."+c+", "+table+"."+c+")")
		}
		b.WriteString(strings.Join(assignments, ", "))
	default:
		// MySQL
		b.WriteString(" ON DUPLICATE KEY UPDATE ")

		assignments := make([]string, 0, len(updateCols)+len(updateCoalesceCols))
		for _, c := range updateCols {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			assignments = append(assignments, c+" = VALUES("+c+")")
		}
		for _, c := range updateCoalesceCols {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			assignments = append(assignments, c+" = COALESCE(VALUES("+c+"), "+c+")")
		}
		b.WriteString(strings.Join(assignments, ", "))
	}

	return db.ExecContext(ctx, b.String(), args...)
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	// We build placeholders using '?' and rely on Dialect.Rebind at execution time.
	if n == 1 {
		return "?"
	}
	var b strings.Builder
	b.Grow(n*2 - 1)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('?')
	}
	return b.String()
}
