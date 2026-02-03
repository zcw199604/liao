package database

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPlaceholders_EdgeCases(t *testing.T) {
	if got := placeholders(0); got != "" {
		t.Fatalf("placeholders(0)=%q, want empty", got)
	}
	if got := placeholders(-1); got != "" {
		t.Fatalf("placeholders(-1)=%q, want empty", got)
	}
	if got := placeholders(1); got != "?" {
		t.Fatalf("placeholders(1)=%q, want ?", got)
	}
	if got := placeholders(3); got != "?, ?, ?" {
		t.Fatalf("placeholders(3)=%q", got)
	}
}

func TestExecUpsert_InsertIgnoreWhenNoUpdateClause_MySQL(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	conn := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`INSERT IGNORE INTO t \(a, b\) VALUES \(\?, \?\)`).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// nil ctx should be treated as background.
	if _, err := ExecUpsert(nil, conn, "t", []string{"a", "b"}, []string{"a"}, nil, nil, 1, 2); err != nil {
		t.Fatalf("ExecUpsert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecUpsert_ErrorGuards(t *testing.T) {
	if _, err := ExecUpsert(context.Background(), nil, "t", []string{"a"}, []string{"a"}, []string{"a"}, nil, 1); err == nil {
		t.Fatalf("expected error for nil db")
	}

	sqlDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	conn := Wrap(sqlDB, PostgresDialect{})

	if _, err := ExecUpsert(context.Background(), conn, "t", nil, []string{"a"}, []string{"a"}, nil, 1); err == nil {
		t.Fatalf("expected error for empty cols")
	}
	if _, err := ExecUpsert(context.Background(), conn, "t", []string{"a", "b"}, []string{"a"}, []string{"b"}, nil, 1); err == nil {
		t.Fatalf("expected error for args mismatch")
	}
	if _, err := ExecUpsert(context.Background(), conn, "t", []string{"a"}, nil, []string{"a"}, nil, 1); err == nil {
		t.Fatalf("expected error for postgres upsert without conflictCols")
	}
}

func TestInsertReturningID_TrimsSpacesAndSemicolon(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	conn := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO foo (a) VALUES (?)")).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(10, 1))

	id, err := InsertReturningID(context.Background(), conn, "  INSERT INTO foo (a) VALUES (?);  ", 1)
	if err != nil {
		t.Fatalf("InsertReturningID: %v", err)
	}
	if id != 10 {
		t.Fatalf("id=%d, want 10", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecInsertIgnore_Postgres_RequiresPlaceholdersRebind(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	conn := Wrap(sqlDB, PostgresDialect{})

	// Ensure our SQL builder uses '?' and relies on dialect.Rebind for $ placeholders.
	want := PostgresDialect{}.Rebind("INSERT INTO t (a) VALUES (?) ON CONFLICT (a) DO NOTHING")
	mock.ExpectExec(regexp.QuoteMeta(want)).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := ExecInsertIgnore(context.Background(), conn, "t", []string{"a"}, []string{"a"}, 1); err != nil {
		t.Fatalf("ExecInsertIgnore: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecUpsert_Postgres_SkipsEmptyUpdateColumns(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	conn := Wrap(sqlDB, PostgresDialect{})

	// Empty update cols should be ignored and not appear as ", ,".
	want := PostgresDialect{}.Rebind("INSERT INTO t (a, b) VALUES (?, ?) ON CONFLICT (a) DO UPDATE SET b = EXCLUDED.b")
	mock.ExpectExec(regexp.QuoteMeta(want)).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := ExecUpsert(context.Background(), conn, "t", []string{"a", "b"}, []string{"a"}, []string{"  ", "b", ""}, []string{" "}, 1, 2); err != nil {
		t.Fatalf("ExecUpsert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecUpsert_MySQL_SkipsEmptyUpdateColumns(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	conn := Wrap(sqlDB, MySQLDialect{})

	want := "INSERT INTO t (a, b) VALUES (?, ?) ON DUPLICATE KEY UPDATE b = VALUES(b)"
	mock.ExpectExec(regexp.QuoteMeta(want)).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := ExecUpsert(context.Background(), conn, "t", []string{"a", "b"}, []string{"a"}, []string{"", "b", "   "}, []string{"  "}, 1, 2); err != nil {
		t.Fatalf("ExecUpsert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecInsertIgnore_ReturnsSqlResult(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	conn := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`INSERT IGNORE INTO t \(a\) VALUES \(\?\)`).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	res, err := ExecInsertIgnore(context.Background(), conn, "t", []string{"a"}, []string{"a"}, 1)
	if err != nil {
		t.Fatalf("ExecInsertIgnore: %v", err)
	}
	if _, err := res.RowsAffected(); err != nil && err != sql.ErrNoRows {
		// sqlmock's result can return ErrNoRows for RowsAffected depending on driver.
		t.Fatalf("RowsAffected: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
