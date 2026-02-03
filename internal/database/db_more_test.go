package database

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// Keep these helpers non-inlined so the wrapper methods show up in coverage.
//go:noinline
func callQueryContext(c Conn, ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.QueryContext(ctx, query, args...)
}

//go:noinline
func callQueryRowContext(c Conn, ctx context.Context, query string, args ...any) *sql.Row {
	return c.QueryRowContext(ctx, query, args...)
}

//go:noinline
func callExecContext(c Conn, ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.ExecContext(ctx, query, args...)
}

func TestDB_HelperMethods(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp),
		sqlmock.MonitorPingsOption(true),
	)
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	if db.SQLDB() != sqlDB {
		t.Fatalf("SQLDB() should return underlying *sql.DB")
	}

	db.SetMaxOpenConns(7)
	if got := db.SQLDB().Stats().MaxOpenConnections; got != 7 {
		t.Fatalf("MaxOpenConnections=%d, want 7", got)
	}

	// There is no public getter for max-idle; call it for coverage.
	db.SetMaxIdleConns(3)

	mock.ExpectPing().WillReturnError(nil)
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("PingContext: %v", err)
	}

	mock.ExpectClose()
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestDB_BeginTx_ReturnsTxWrapperAndCommit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, PostgresDialect{})

	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	var conn Conn = tx
	if conn.Dialect().Name() != "postgres" {
		t.Fatalf("tx dialect=%q, want postgres", conn.Dialect().Name())
	}
	if tx.SQLTx() == nil {
		t.Fatalf("SQLTx() should not be nil")
	}

	wantExec := PostgresDialect{}.Rebind("INSERT INTO t (a) VALUES (?)")
	mock.ExpectExec(regexp.QuoteMeta(wantExec)).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := callExecContext(conn, context.Background(), "INSERT INTO t (a) VALUES (?)", 1); err != nil {
		t.Fatalf("ExecContext: %v", err)
	}

	wantQuery := PostgresDialect{}.Rebind("SELECT ? AS v")
	mock.ExpectQuery(regexp.QuoteMeta(wantQuery)).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"v"}).AddRow(2))
	rows, err := callQueryContext(conn, context.Background(), "SELECT ? AS v", 2)
	if err != nil {
		t.Fatalf("QueryContext: %v", err)
	}
	rows.Close()

	mock.ExpectQuery(regexp.QuoteMeta(wantQuery)).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"v"}).AddRow(3))
	row := callQueryRowContext(conn, context.Background(), "SELECT ? AS v", 3)
	var v int
	if err := row.Scan(&v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if v != 3 {
		t.Fatalf("v=%d, want 3", v)
	}

	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestDB_BeginTx_Rollback(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestDB_BeginTx_Error(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	mock.ExpectBegin().WillReturnError(sql.ErrConnDone)
	if _, err := db.BeginTx(context.Background(), nil); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestDB_QueryContext_RebindsForPostgres(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, PostgresDialect{})
	var conn Conn = db

	wantQuery := PostgresDialect{}.Rebind("SELECT ? AS v")
	mock.ExpectQuery(regexp.QuoteMeta(wantQuery)).
		WithArgs(123).
		WillReturnRows(sqlmock.NewRows([]string{"v"}).AddRow(123))

	rows, err := callQueryContext(conn, context.Background(), "SELECT ? AS v", 123)
	if err != nil {
		t.Fatalf("QueryContext: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatalf("expected one row")
	}
	var v int
	if err := rows.Scan(&v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if v != 123 {
		t.Fatalf("v = %d, want 123", v)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestDB_QueryRowContext_RebindsForPostgres(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, PostgresDialect{})
	var conn Conn = db

	wantQuery := PostgresDialect{}.Rebind("SELECT ? AS v")
	mock.ExpectQuery(regexp.QuoteMeta(wantQuery)).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"v"}).AddRow(7))

	row := callQueryRowContext(conn, context.Background(), "SELECT ? AS v", 7)
	var v int
	if err := row.Scan(&v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if v != 7 {
		t.Fatalf("v = %d, want 7", v)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
