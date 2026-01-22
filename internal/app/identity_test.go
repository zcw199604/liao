package app

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestIdentityService_GetAll_FormatsTimes(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "name", "sex", "created_at", "last_used_at"}).
		AddRow("a", "A", "男", now, now).
		AddRow("b", "B", "女", now, sql.NullTime{Valid: false})

	mock.ExpectQuery(`SELECT id, name, sex, created_at, last_used_at FROM identity ORDER BY last_used_at DESC`).
		WillReturnRows(rows)

	svc := NewIdentityService(db)
	list, err := svc.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d, want 2", len(list))
	}
	if list[0].ID != "a" || list[0].LastUsedAt == "" {
		t.Fatalf("unexpected first: %+v", list[0])
	}
	if list[1].ID != "b" || list[1].LastUsedAt != "" {
		t.Fatalf("unexpected second: %+v", list[1])
	}
}

func TestIdentityService_GetByID_NotFound(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}))

	svc := NewIdentityService(db)
	got, err := svc.GetByID(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil")
	}
}

func TestIdentityService_Create_Insert(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO identity`).
		WithArgs(sqlmock.AnyArg(), "Alice", "女", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc := NewIdentityService(db)
	created, err := svc.Create(context.Background(), "Alice", "女")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created == nil || created.ID == "" || created.Name != "Alice" || created.Sex != "女" {
		t.Fatalf("unexpected: %+v", created)
	}
}

func TestIdentityService_UpdateID_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Old", "男", now, now))
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("new").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}))

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM identity WHERE id = \?`).WithArgs("old").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO identity`).
		WithArgs("new", "New", "女", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	svc := NewIdentityService(db)
	updated, err := svc.UpdateID(context.Background(), "old", "new", "New", "女")
	if err != nil {
		t.Fatalf("UpdateID: %v", err)
	}
	if updated == nil || updated.ID != "new" || updated.Name != "New" || updated.Sex != "女" {
		t.Fatalf("unexpected: %+v", updated)
	}
}
