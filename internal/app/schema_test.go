package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/database"
)

func TestEnsureSchema_NilDB(t *testing.T) {
	if err := ensureSchema(nil); err == nil {
		t.Fatalf("expected error")
	}
}

func TestEnsureSchema_RunsMigrator(t *testing.T) {
	tmp := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	mDir := filepath.Join(tmp, "sql", "mysql")
	if err := os.MkdirAll(mDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mDir, "001_init.sql"), []byte("CREATE TABLE IF NOT EXISTS t (id INT);"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	wrapped := database.Wrap(db, database.MySQLDialect{})

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS t`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("001_init", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := ensureSchema(wrapped); err != nil {
		t.Fatalf("ensureSchema: %v", err)
	}
}

func TestEnsureSchema_ReturnsMigratorError(t *testing.T) {
	tmp := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	mDir := filepath.Join(tmp, "sql", "mysql")
	if err := os.MkdirAll(mDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mDir, "001_init.sql"), []byte("CREATE TABLE IF NOT EXISTS t (id INT);"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	wrapped := database.Wrap(db, database.MySQLDialect{})

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS t`).
		WillReturnError(errors.New("exec fail"))

	if err := ensureSchema(wrapped); err == nil || !strings.Contains(err.Error(), "数据库迁移失败") {
		t.Fatalf("err=%v", err)
	}
}
