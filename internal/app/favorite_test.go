package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestFavoriteService_Add_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnError(sql.ErrConnDone)

	svc := NewFavoriteService(db)
	if _, err := svc.Add(context.Background(), "i1", "u1", "Bob"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFavoriteService_Add_ReturnsExisting(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 10, 12, 0, 0, 0, time.Local)
	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}).
			AddRow(int64(1), "i1", "u1", sql.NullString{String: "Bob", Valid: true}, sql.NullTime{Time: createTime, Valid: true}))

	svc := NewFavoriteService(db)
	fav, err := svc.Add(context.Background(), "i1", "u1", "Bob")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if fav == nil || fav.ID != 1 || fav.IdentityID != "i1" || fav.TargetUserID != "u1" || fav.TargetUserName != "Bob" {
		t.Fatalf("unexpected fav: %+v", fav)
	}
}

func TestFavoriteService_Add_InsertsWhenMissing(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}))

	mock.ExpectExec(`INSERT INTO chat_favorites \(identity_id, target_user_id, target_user_name, create_time\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs("i1", "u1", "Bob", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(7, 1))

	svc := NewFavoriteService(db)
	fav, err := svc.Add(context.Background(), "i1", "u1", "Bob")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if fav == nil || fav.ID != 7 || fav.TargetUserName != "Bob" || fav.CreateTime == "" {
		t.Fatalf("unexpected fav: %+v", fav)
	}
}

func TestFavoriteService_Add_UsesNullForEmptyName(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}))

	mock.ExpectExec(`INSERT INTO chat_favorites \(identity_id, target_user_id, target_user_name, create_time\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs("i1", "u1", nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(7, 1))

	svc := NewFavoriteService(db)
	fav, err := svc.Add(context.Background(), "i1", "u1", "   ")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if fav == nil || fav.TargetUserName != "   " {
		t.Fatalf("unexpected fav: %+v", fav)
	}
}

func TestFavoriteService_Add_InsertError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}))

	mock.ExpectExec(`INSERT INTO chat_favorites \(identity_id, target_user_id, target_user_name, create_time\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs("i1", "u1", "Bob", sqlmock.AnyArg()).
		WillReturnError(errors.New("insert fail"))

	svc := NewFavoriteService(db)
	if _, err := svc.Add(context.Background(), "i1", "u1", "Bob"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFavoriteService_ListAll_Errors(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites ORDER BY create_time DESC`).
		WillReturnError(errors.New("query fail"))

	svc := NewFavoriteService(db)
	if _, err := svc.ListAll(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFavoriteService_ListAll_ScanError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites ORDER BY create_time DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}).
			AddRow("not-int", "i1", "u1", sql.NullString{String: "Bob", Valid: true}, sql.NullTime{Valid: false}))

	svc := NewFavoriteService(db)
	if _, err := svc.ListAll(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFavoriteService_ListAll_TargetNameAndCreateTimeNull(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites ORDER BY create_time DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}).
			AddRow(int64(1), "i1", "u1", sql.NullString{Valid: false}, sql.NullTime{Valid: false}))

	svc := NewFavoriteService(db)
	list, err := svc.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len=%d, want 1", len(list))
	}
	if list[0].TargetUserName != "" {
		t.Fatalf("TargetUserName=%q, want empty", list[0].TargetUserName)
	}
	if list[0].CreateTime != "" {
		t.Fatalf("CreateTime=%q, want empty", list[0].CreateTime)
	}
}

func TestFavoriteService_IsFavorite(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT 1 FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	svc := NewFavoriteService(db)
	ok, err := svc.IsFavorite(context.Background(), "i1", "u1")
	if err != nil {
		t.Fatalf("IsFavorite failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected favorite=true")
	}

	mock.ExpectQuery(`SELECT 1 FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u2").
		WillReturnRows(sqlmock.NewRows([]string{"1"}))

	ok, err = svc.IsFavorite(context.Background(), "i1", "u2")
	if err != nil {
		t.Fatalf("IsFavorite failed: %v", err)
	}
	if ok {
		t.Fatalf("expected favorite=false")
	}
}

func TestFavoriteService_IsFavorite_DBError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT 1 FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnError(sql.ErrConnDone)

	svc := NewFavoriteService(db)
	if _, err := svc.IsFavorite(context.Background(), "i1", "u1"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFavoriteService_RemoveByID(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM chat_favorites WHERE id = \?`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := NewFavoriteService(db)
	if err := svc.RemoveByID(context.Background(), 1); err != nil {
		t.Fatalf("RemoveByID failed: %v", err)
	}
}

func TestFavoriteService_FindByIdentityAndTarget_ErrorsAndNullFields(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u1").
		WillReturnError(sql.ErrConnDone)

	svc := NewFavoriteService(db)
	if _, err := svc.findByIdentityAndTarget(context.Background(), "i1", "u1"); err == nil {
		t.Fatalf("expected error")
	}

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u2").
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}))

	fav, err := svc.findByIdentityAndTarget(context.Background(), "i1", "u2")
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if fav != nil {
		t.Fatalf("expected nil, got %+v", fav)
	}

	mock.ExpectQuery(`SELECT id, identity_id, target_user_id, target_user_name, create_time FROM chat_favorites WHERE identity_id = \? AND target_user_id = \? LIMIT 1`).
		WithArgs("i1", "u3").
		WillReturnRows(sqlmock.NewRows([]string{"id", "identity_id", "target_user_id", "target_user_name", "create_time"}).
			AddRow(int64(1), "i1", "u3", sql.NullString{Valid: false}, sql.NullTime{Valid: false}))

	fav, err = svc.findByIdentityAndTarget(context.Background(), "i1", "u3")
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if fav == nil {
		t.Fatalf("expected fav")
	}
	if fav.TargetUserName != "" {
		t.Fatalf("TargetUserName=%q, want empty", fav.TargetUserName)
	}
	if fav.CreateTime != "" {
		t.Fatalf("CreateTime=%q, want empty", fav.CreateTime)
	}
}
