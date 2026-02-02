package database

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresDialect implements Dialect for PostgreSQL.
type PostgresDialect struct{}

func (PostgresDialect) Name() string { return "postgres" }

func (PostgresDialect) DriverName() string { return "pgx" }

func (PostgresDialect) Rebind(query string) string { return RebindDollar(query) }

func (PostgresDialect) IsDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	// 23505: unique_violation
	return errors.As(err, &pgErr) && pgErr != nil && pgErr.Code == "23505"
}

func (PostgresDialect) IsDuplicateColumn(err error) bool {
	var pgErr *pgconn.PgError
	// 42701: duplicate_column
	return errors.As(err, &pgErr) && pgErr != nil && pgErr.Code == "42701"
}

func (PostgresDialect) IsDuplicateIndex(err error) bool {
	var pgErr *pgconn.PgError
	// 42P07: duplicate_table (covers some duplicate object cases)
	// 42710: duplicate_object
	// Index creation failures can vary; we treat both as "already exists".
	return errors.As(err, &pgErr) && pgErr != nil && (pgErr.Code == "42P07" || pgErr.Code == "42710")
}
