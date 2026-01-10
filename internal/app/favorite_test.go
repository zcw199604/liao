package app

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

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

