package database

import (
	"fmt"
	"strings"
)

// Dialect defines database-specific behaviors that leak through database/sql:
// - placeholder syntax (? vs $1)
// - driver name
// - common error codes (duplicate key, etc.)
//
// Keep this interface small; only add methods when we have a real cross-dialect need.
type Dialect interface {
	// Name returns a normalized dialect name ("mysql" or "postgres").
	Name() string
	// DriverName returns the database/sql driver name ("mysql" or "pgx").
	DriverName() string
	// Rebind converts a query written with '?' placeholders into the dialect-specific form.
	// For MySQL it returns the input unchanged; for Postgres it converts to $1, $2, ...
	Rebind(query string) string

	// IsDuplicateKey reports whether err indicates a unique/primary key conflict.
	IsDuplicateKey(err error) bool
	// IsDuplicateColumn reports whether err indicates a duplicate column during DDL migration.
	IsDuplicateColumn(err error) bool
	// IsDuplicateIndex reports whether err indicates a duplicate index/key during DDL migration.
	IsDuplicateIndex(err error) bool
}

// DialectFromScheme returns the dialect for a DB_URL scheme.
// Supported schemes:
// - mysql
// - postgres
// - postgresql
func DialectFromScheme(scheme string) (Dialect, error) {
	s := strings.ToLower(strings.TrimSpace(scheme))
	switch s {
	case "mysql":
		return MySQLDialect{}, nil
	case "postgres", "postgresql":
		return PostgresDialect{}, nil
	default:
		return nil, fmt.Errorf("unsupported DB_URL scheme: %s", scheme)
	}
}
