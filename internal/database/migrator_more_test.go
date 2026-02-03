package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

func TestPostgresDialect_DriverName(t *testing.T) {
	if got := (PostgresDialect{}).DriverName(); got != "pgx" {
		t.Fatalf("DriverName() = %q, want %q", got, "pgx")
	}
}

func TestNewMigrator_DefaultBaseDirWhenBlank(t *testing.T) {
	m := NewMigrator("   ")
	if m.baseDir != "sql" {
		t.Fatalf("baseDir = %q, want %q", m.baseDir, "sql")
	}
}

func TestMigrator_Migrate_NilDB(t *testing.T) {
	m := NewMigrator("sql")
	if err := m.Migrate(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil db")
	}
}

func TestMigrator_Migrate_NilContextUsesBackground(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	mysqlDir := filepath.Join(baseDir, "mysql")
	if err := os.MkdirAll(mysqlDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "001_init.sql"), []byte(`CREATE TABLE IF NOT EXISTS t (id INT);`), 0o644); err != nil {
		t.Fatalf("write 001: %v", err)
	}

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS t`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("001_init", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	m := NewMigrator(baseDir)
	if err := m.Migrate(nil, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_MigrationsDirNotFound(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	// Intentionally do NOT create sql/mysql directory.

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	m := NewMigrator(baseDir)
	err = m.Migrate(context.Background(), db)
	if err == nil {
		t.Fatalf("expected error")
	}
	want := "migrations directory not found: " + filepath.Join(baseDir, "mysql")
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %q, want contains %q", err.Error(), want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_ToleratesDuplicateColumnAndStillRecordsVersion(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	mysqlDir := filepath.Join(baseDir, "mysql")
	if err := os.MkdirAll(mysqlDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "001_alter.sql"), []byte(`ALTER TABLE t ADD COLUMN name varchar(10);`), 0o644); err != nil {
		t.Fatalf("write 001: %v", err)
	}

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	// Duplicate column error should be tolerated and still record the version.
	mock.ExpectExec(`ALTER TABLE t ADD COLUMN name`).
		WillReturnError(&mysql.MySQLError{Number: 1060, Message: "dup col"})
	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("001_alter", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestEnsureMigrationsTable_ExecError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnError(errors.New("boom"))

	err = ensureMigrationsTable(context.Background(), db)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "create schema_migrations") {
		t.Fatalf("err = %q, want wrapped create schema_migrations", err.Error())
	}
}

func TestRecordAppliedVersion_EmptyVersion(t *testing.T) {
	sqlDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})
	err = recordAppliedVersion(context.Background(), db, "   ")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "empty migration version") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "empty migration version")
	}
}

func TestRecordAppliedVersion_DuplicateKeyIsTreatedAsApplied(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("001_init", sqlmock.AnyArg()).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "dup key"})

	if err := recordAppliedVersion(context.Background(), db, "001_init"); err != nil {
		t.Fatalf("recordAppliedVersion: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestSplitSQLStatements_RespectsQuotesBackticksAndComments(t *testing.T) {
	sqlText := "" +
		"CREATE TABLE t (name TEXT DEFAULT 'a;''b');" +
		"INSERT INTO t (name) VALUES (\"x;\"\"y\");" +
		"INSERT INTO `t;name` VALUES (1);" +
		"-- comment with; semicolon\n" +
		"SELECT 1;" +
		"/* block; comment */SELECT 2"

	got := splitSQLStatements(sqlText)
	want := []string{
		"CREATE TABLE t (name TEXT DEFAULT 'a;''b')",
		"INSERT INTO t (name) VALUES (\"x;\"\"y\")",
		"INSERT INTO `t;name` VALUES (1)",
		"-- comment with; semicolon\nSELECT 1",
		"/* block; comment */SELECT 2",
	}

	if len(got) != len(want) {
		t.Fatalf("len(statements) = %d, want %d\n%q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("statements[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadAppliedVersions_QueryError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnError(errors.New("boom"))

	if _, err := loadAppliedVersions(context.Background(), db); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestLoadAppliedVersions_ScanError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	// Return a resultset that doesn't match the Scan destination count.
	// rows.Scan(&v) should fail because the row has 2 columns.
	rows := sqlmock.NewRows([]string{"version", "extra"}).AddRow("001", "x")
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(rows)

	if _, err := loadAppliedVersions(context.Background(), db); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRecordAppliedVersion_ExecErrorIsReturned(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("001_init", sqlmock.AnyArg()).
		WillReturnError(errors.New("boom"))

	if err := recordAppliedVersion(context.Background(), db, "001_init"); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestLoadAppliedVersions_RowsErr(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})

	rows := sqlmock.NewRows([]string{"version"}).AddRow("001").RowError(0, errors.New("next boom"))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(rows)

	if _, err := loadAppliedVersions(context.Background(), db); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_EnsureMigrationsTableError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnError(errors.New("boom"))

	m := NewMigrator("sql")
	if err := m.Migrate(context.Background(), db); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_LoadAppliedVersionsError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnError(errors.New("boom"))

	m := NewMigrator("sql")
	if err := m.Migrate(context.Background(), db); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_ReadDirNotDirectory(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Create a file where the migrations directory should be.
	if err := os.WriteFile(filepath.Join(baseDir, "mysql"), []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_SkipsSubdirsAndNonSqlFiles(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	mysqlDir := filepath.Join(baseDir, "mysql")
	if err := os.MkdirAll(filepath.Join(mysqlDir, "dir1"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_ToleratesDuplicateIndex(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	mysqlDir := filepath.Join(baseDir, "mysql")
	if err := os.MkdirAll(mysqlDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "001_idx.sql"), []byte(`CREATE INDEX idx1 ON t (a);`), 0o644); err != nil {
		t.Fatalf("write 001: %v", err)
	}

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	mock.ExpectExec(`CREATE INDEX idx1 ON t`).
		WillReturnError(&mysql.MySQLError{Number: 1061, Message: "dup idx"})
	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("001_idx", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_ExecErrorPropagates(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	mysqlDir := filepath.Join(baseDir, "mysql")
	if err := os.MkdirAll(mysqlDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "001_bad.sql"), []byte(`CREATE TABLE t (id INT);`), 0o644); err != nil {
		t.Fatalf("write 001: %v", err)
	}

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	mock.ExpectExec(`CREATE TABLE t`).
		WillReturnError(errors.New("boom"))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_RecordAppliedVersionErrorPropagates(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	mysqlDir := filepath.Join(baseDir, "mysql")
	if err := os.MkdirAll(mysqlDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "001_ok.sql"), []byte(`CREATE TABLE t (id INT);`), 0o644); err != nil {
		t.Fatalf("write 001: %v", err)
	}

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	mock.ExpectExec(`CREATE TABLE t`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("001_ok", sqlmock.AnyArg()).
		WillReturnError(errors.New("boom"))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Migrate_ReadFileError(t *testing.T) {
	// chmod(0) is unreliable on Windows; keep the suite portable.
	if runtime.GOOS == "windows" {
		t.Skip("skip chmod-based unreadable file test on windows")
	}

	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	mysqlDir := filepath.Join(baseDir, "mysql")
	if err := os.MkdirAll(mysqlDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	p := filepath.Join(mysqlDir, "001_unreadable.sql")
	if err := os.WriteFile(p, []byte(`CREATE TABLE t (id INT);`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(p, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, MySQLDialect{})
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err == nil {
		t.Fatalf("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
