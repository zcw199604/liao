package app

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestEnsureSchema_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// ensureSchema 内部固定有多条 CREATE TABLE IF NOT EXISTS ...
	for i := 0; i < 15; i++ {
		mock.ExpectExec(`CREATE TABLE IF NOT EXISTS`).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	if err := ensureSchema(db); err != nil {
		t.Fatalf("ensureSchema: %v", err)
	}
}

func TestEnsureSchema_ExecError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS`).
		WillReturnError(errors.New("exec fail"))

	if err := ensureSchema(db); err == nil {
		t.Fatalf("expected error")
	}
}
