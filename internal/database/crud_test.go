package database

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestInsertReturningID_MySQL(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	conn := Wrap(db, MySQLDialect{})
	mock.ExpectExec(`INSERT INTO foo`).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(9, 1))

	got, err := InsertReturningID(context.Background(), conn, "INSERT INTO foo (a) VALUES (?)", 1)
	if err != nil {
		t.Fatalf("InsertReturningID: %v", err)
	}
	if got != 9 {
		t.Fatalf("id=%d", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestInsertReturningID_Postgres(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	conn := Wrap(db, PostgresDialect{})

	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(9))
	mock.ExpectQuery(`INSERT INTO foo \(a\) VALUES \(\$1\) RETURNING id`).
		WithArgs(1).
		WillReturnRows(rows)

	got, err := InsertReturningID(context.Background(), conn, "INSERT INTO foo (a) VALUES (?)", 1)
	if err != nil {
		t.Fatalf("InsertReturningID: %v", err)
	}
	if got != 9 {
		t.Fatalf("id=%d", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecInsertIgnore_MySQL(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	conn := Wrap(db, MySQLDialect{})
	mock.ExpectExec(`INSERT IGNORE INTO t \(a, b\) VALUES \(\?, \?\)`).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := ExecInsertIgnore(context.Background(), conn, "t", []string{"a", "b"}, []string{"a"}, 1, 2); err != nil {
		t.Fatalf("ExecInsertIgnore: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecInsertIgnore_Postgres(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	conn := Wrap(db, PostgresDialect{})
	mock.ExpectExec(`INSERT INTO t \(a, b\) VALUES \(\$1, \$2\) ON CONFLICT \(a\) DO NOTHING`).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := ExecInsertIgnore(context.Background(), conn, "t", []string{"a", "b"}, []string{"a"}, 1, 2); err != nil {
		t.Fatalf("ExecInsertIgnore: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecUpsert_MySQL_Replace(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	conn := Wrap(db, MySQLDialect{})
	mock.ExpectExec(`INSERT INTO t \(a, b\) VALUES \(\?, \?\) ON DUPLICATE KEY UPDATE b = VALUES\(b\)`).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := ExecUpsert(context.Background(), conn, "t", []string{"a", "b"}, []string{"a"}, []string{"b"}, nil, 1, 2); err != nil {
		t.Fatalf("ExecUpsert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecUpsert_MySQL_Coalesce(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	conn := Wrap(db, MySQLDialect{})
	mock.ExpectExec(`INSERT INTO t \(a, b\) VALUES \(\?, \?\) ON DUPLICATE KEY UPDATE b = COALESCE\(VALUES\(b\), b\)`).
		WithArgs(1, nil).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := ExecUpsert(context.Background(), conn, "t", []string{"a", "b"}, []string{"a"}, nil, []string{"b"}, 1, nil); err != nil {
		t.Fatalf("ExecUpsert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecUpsert_Postgres_Replace(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	conn := Wrap(db, PostgresDialect{})
	mock.ExpectExec(`INSERT INTO t \(a, b\) VALUES \(\$1, \$2\) ON CONFLICT \(a\) DO UPDATE SET b = EXCLUDED\.b`).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := ExecUpsert(context.Background(), conn, "t", []string{"a", "b"}, []string{"a"}, []string{"b"}, nil, 1, 2); err != nil {
		t.Fatalf("ExecUpsert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExecUpsert_Postgres_Coalesce(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	conn := Wrap(db, PostgresDialect{})
	mock.ExpectExec(`INSERT INTO t \(a, b\) VALUES \(\$1, \$2\) ON CONFLICT \(a\) DO UPDATE SET b = COALESCE\(EXCLUDED\.b, t\.b\)`).
		WithArgs(1, nil).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := ExecUpsert(context.Background(), conn, "t", []string{"a", "b"}, []string{"a"}, nil, []string{"b"}, 1, nil); err != nil {
		t.Fatalf("ExecUpsert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
