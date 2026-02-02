package database

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMigrator_MySQL_AppliesMigrationsInOrderAndRecords(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	mysqlDir := filepath.Join(baseDir, "mysql")
	if err := os.MkdirAll(mysqlDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "001_init.sql"), []byte(`
		CREATE TABLE IF NOT EXISTS t (id INT);
		CREATE TABLE IF NOT EXISTS u (id INT);
	`), 0o644); err != nil {
		t.Fatalf("write 001: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "002_alter.sql"), []byte(`
		ALTER TABLE t ADD COLUMN name varchar(10);
	`), 0o644); err != nil {
		t.Fatalf("write 002: %v", err)
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
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS u`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("001_init", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(`ALTER TABLE t ADD COLUMN name`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("002_alter", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_SkipsAlreadyAppliedVersions(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	mysqlDir := filepath.Join(baseDir, "mysql")
	if err := os.MkdirAll(mysqlDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "001_init.sql"), []byte(`CREATE TABLE IF NOT EXISTS t (id INT);`), 0o644); err != nil {
		t.Fatalf("write 001: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mysqlDir, "002_alter.sql"), []byte(`ALTER TABLE t ADD COLUMN name varchar(10);`), 0o644); err != nil {
		t.Fatalf("write 002: %v", err)
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
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("001_init"))

	// 001_init is already applied, so only 002 runs.
	mock.ExpectExec(`ALTER TABLE t ADD COLUMN name`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("002_alter", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrator_Postgres_RebindsInsertPlaceholders(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "sql")
	pgDir := filepath.Join(baseDir, "postgres")
	if err := os.MkdirAll(pgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pgDir, "001_init.sql"), []byte(`CREATE TABLE IF NOT EXISTS t (id int);`), 0o644); err != nil {
		t.Fatalf("write 001: %v", err)
	}

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := Wrap(sqlDB, PostgresDialect{})

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS t`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	wantInsert := PostgresDialect{}.Rebind("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)")
	mock.ExpectExec(regexp.QuoteMeta(wantInsert)).
		WithArgs("001_init", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	m := NewMigrator(baseDir)
	if err := m.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
