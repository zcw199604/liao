package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestIdentityService_GetAll_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, name, sex, created_at, last_used_at FROM identity ORDER BY last_used_at DESC`).
		WillReturnError(errors.New("query fail"))

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.GetAll(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIdentityService_GetAll_ScanError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "name", "sex", "created_at", "last_used_at"}).
		AddRow("a", "A", "男", true, now)

	mock.ExpectQuery(`SELECT id, name, sex, created_at, last_used_at FROM identity ORDER BY last_used_at DESC`).
		WillReturnRows(rows)

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.GetAll(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIdentityService_GetAll_FormatsTimes(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "name", "sex", "created_at", "last_used_at"}).
		AddRow("a", "A", "男", now, now).
		AddRow("b", "B", "女", now, sql.NullTime{Valid: false})

	mock.ExpectQuery(`SELECT id, name, sex, created_at, last_used_at FROM identity ORDER BY last_used_at DESC`).
		WillReturnRows(rows)

	svc := NewIdentityService(wrapMySQLDB(db))
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

	svc := NewIdentityService(wrapMySQLDB(db))
	got, err := svc.GetByID(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil")
	}
}

func TestIdentityService_GetByID_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Alice", "女", now, now))

	svc := NewIdentityService(wrapMySQLDB(db))
	got, err := svc.GetByID(context.Background(), "id1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID != "id1" || got.Name != "Alice" || got.Sex != "女" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestIdentityService_GetByID_DBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnError(sql.ErrConnDone)

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.GetByID(context.Background(), "id1"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIdentityService_Create_Insert(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO identity`).
		WithArgs(sqlmock.AnyArg(), "Alice", "女", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc := NewIdentityService(wrapMySQLDB(db))
	created, err := svc.Create(context.Background(), "Alice", "女")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created == nil || created.ID == "" || created.Name != "Alice" || created.Sex != "女" {
		t.Fatalf("unexpected: %+v", created)
	}
}

func TestIdentityService_QuickCreate_SexBranchesAndCreateError(t *testing.T) {
	old := identityRandIntnFn
	t.Cleanup(func() { identityRandIntnFn = old })

	// 男
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		identityRandIntnFn = func(n int) int { return 0 }
		mock.ExpectExec(`INSERT INTO identity`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "男", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		svc := NewIdentityService(wrapMySQLDB(db))
		created, err := svc.QuickCreate(context.Background())
		if err != nil {
			t.Fatalf("QuickCreate: %v", err)
		}
		if created == nil || created.Sex != "男" {
			t.Fatalf("unexpected: %+v", created)
		}
	}

	// 女
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		identityRandIntnFn = func(n int) int { return 1 }
		mock.ExpectExec(`INSERT INTO identity`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "女", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		svc := NewIdentityService(wrapMySQLDB(db))
		created, err := svc.QuickCreate(context.Background())
		if err != nil {
			t.Fatalf("QuickCreate: %v", err)
		}
		if created == nil || created.Sex != "女" {
			t.Fatalf("unexpected: %+v", created)
		}
	}

	// Create error
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		identityRandIntnFn = func(n int) int { return 1 }
		mock.ExpectExec(`INSERT INTO identity`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "女", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert fail"))

		svc := NewIdentityService(wrapMySQLDB(db))
		if _, err := svc.QuickCreate(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestIdentityService_Update_ErrorsAndNotFound(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnError(sql.ErrConnDone)

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.Update(context.Background(), "id1", "New", "女"); err == nil {
		t.Fatalf("expected error")
	}

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}))

	got, err := svc.Update(context.Background(), "missing", "New", "女")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil")
	}
}

func TestIdentityService_Update_ExecError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Old", "男", now, now))

	mock.ExpectExec(`UPDATE identity SET name = \?, sex = \?, last_used_at = \? WHERE id = \?`).
		WithArgs("New", "女", sqlmock.AnyArg(), "id1").
		WillReturnError(errors.New("update fail"))

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.Update(context.Background(), "id1", "New", "女"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIdentityService_Delete_DBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM identity WHERE id = \?`).
		WithArgs("id1").
		WillReturnError(errors.New("delete fail"))

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.Delete(context.Background(), "id1"); err == nil {
		t.Fatalf("expected error")
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

	svc := NewIdentityService(wrapMySQLDB(db))
	updated, err := svc.UpdateID(context.Background(), "old", "new", "New", "女")
	if err != nil {
		t.Fatalf("UpdateID: %v", err)
	}
	if updated == nil || updated.ID != "new" || updated.Name != "New" || updated.Sex != "女" {
		t.Fatalf("unexpected: %+v", updated)
	}
}

func TestIdentityService_UpdateID_GetByIDErrors(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnError(sql.ErrConnDone)

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.UpdateID(context.Background(), "old", "new", "New", "女"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIdentityService_UpdateID_NewIDLookupError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Old", "男", now, now))
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("new").
		WillReturnError(sql.ErrConnDone)

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.UpdateID(context.Background(), "old", "new", "New", "女"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIdentityService_UpdateID_OldNotFound(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}))

	svc := NewIdentityService(wrapMySQLDB(db))
	got, err := svc.UpdateID(context.Background(), "old", "new", "New", "女")
	if err != nil {
		t.Fatalf("UpdateID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil")
	}
}

func TestIdentityService_UpdateID_BeginTxError(t *testing.T) {
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

	mock.ExpectBegin().WillReturnError(errors.New("begin fail"))

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.UpdateID(context.Background(), "old", "new", "New", "女"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIdentityService_UpdateID_DeleteError(t *testing.T) {
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
	mock.ExpectExec(`DELETE FROM identity WHERE id = \?`).WithArgs("old").WillReturnError(errors.New("delete fail"))
	mock.ExpectRollback()

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.UpdateID(context.Background(), "old", "new", "New", "女"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIdentityService_UpdateID_CommitError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createdAt := time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC)
	createdAtStr := createdAt.Format("2006-01-02 15:04:05")

	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("old").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}).
			AddRow("Old", "男", createdAt, createdAt))
	mock.ExpectQuery(`SELECT name, sex, created_at, last_used_at FROM identity WHERE id = \?`).
		WithArgs("new").
		WillReturnRows(sqlmock.NewRows([]string{"name", "sex", "created_at", "last_used_at"}))

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM identity WHERE id = \?`).WithArgs("old").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO identity`).
		WithArgs("new", "New", "女", createdAtStr, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit fail"))

	svc := NewIdentityService(wrapMySQLDB(db))
	if _, err := svc.UpdateID(context.Background(), "old", "new", "New", "女"); err == nil {
		t.Fatalf("expected error")
	}
}
