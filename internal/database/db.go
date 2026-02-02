package database

import (
	"context"
	"database/sql"
)

// Conn is the common subset used across *sql.DB and *sql.Tx style operations.
// Both DB and Tx wrappers implement it so helpers can work in/out of transactions.
type Conn interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Dialect() Dialect
}

// DB wraps *sql.DB and applies Dialect.Rebind for every query.
type DB struct {
	db      *sql.DB
	dialect Dialect
}

func Wrap(db *sql.DB, dialect Dialect) *DB {
	return &DB{db: db, dialect: dialect}
}

func (d *DB) Dialect() Dialect { return d.dialect }

func (d *DB) SQLDB() *sql.DB { return d.db }

func (d *DB) Close() error { return d.db.Close() }

func (d *DB) PingContext(ctx context.Context) error { return d.db.PingContext(ctx) }

func (d *DB) SetMaxOpenConns(n int) { d.db.SetMaxOpenConns(n) }

func (d *DB) SetMaxIdleConns(n int) { d.db.SetMaxIdleConns(n) }

func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.db.ExecContext(ctx, d.dialect.Rebind(query), args...)
}

func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, d.dialect.Rebind(query), args...)
}

func (d *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, d.dialect.Rebind(query), args...)
}

func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := d.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, dialect: d.dialect}, nil
}

// Tx wraps *sql.Tx and applies Dialect.Rebind for every query.
type Tx struct {
	tx      *sql.Tx
	dialect Dialect
}

func (t *Tx) Dialect() Dialect { return t.dialect }

func (t *Tx) SQLTx() *sql.Tx { return t.tx }

func (t *Tx) Commit() error { return t.tx.Commit() }

func (t *Tx) Rollback() error { return t.tx.Rollback() }

func (t *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, t.dialect.Rebind(query), args...)
}

func (t *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, t.dialect.Rebind(query), args...)
}

func (t *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, t.dialect.Rebind(query), args...)
}
