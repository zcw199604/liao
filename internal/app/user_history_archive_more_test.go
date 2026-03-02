package app

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

type stubUserArchiveService struct {
	persistCalls int
	lastOwner    string
	lastUsers    []map[string]any
	lastSource   UserArchiveListSource
}

func (s *stubUserArchiveService) PersistUserList(_ context.Context, ownerUserID string, users []map[string]any, source UserArchiveListSource) {
	s.persistCalls++
	s.lastOwner = ownerUserID
	s.lastUsers = users
	s.lastSource = source
}

func (s *stubUserArchiveService) MergeArchivedUsers(_ context.Context, _ string, upstream []map[string]any, _ UserArchiveListSource) []map[string]any {
	return upstream
}

func (s *stubUserArchiveService) TouchConversation(_ context.Context, _, _ string) {}

func (s *stubUserArchiveService) SaveLastMessage(_ context.Context, _, _, _, _ string) {}

func (s *stubUserArchiveService) DeleteConversation(_ context.Context, _, _ string) {}

func TestPersistArchivedUserListAndCloneUserList(t *testing.T) {
	var nilApp *App
	if got := nilApp.persistArchivedUserList("me", []map[string]any{{"id": "u1"}}, UserArchiveListSourceHistory); got != -1 {
		t.Fatalf("nil app should return -1, got %d", got)
	}

	appNoArchive := &App{}
	if got := appNoArchive.persistArchivedUserList("me", []map[string]any{{"id": "u1"}}, UserArchiveListSourceHistory); got != -1 {
		t.Fatalf("app without archive should return -1, got %d", got)
	}

	stub := &stubUserArchiveService{}
	app := &App{userArchive: stub}
	if got := app.persistArchivedUserList(" ", []map[string]any{{"id": "u1"}}, UserArchiveListSourceHistory); got != -1 {
		t.Fatalf("blank owner should return -1, got %d", got)
	}
	if got := app.persistArchivedUserList("me", nil, UserArchiveListSourceHistory); got != -1 {
		t.Fatalf("empty users should return -1, got %d", got)
	}

	users := []map[string]any{{"id": "u1", "name": "n1"}}
	if got := app.persistArchivedUserList("me", users, UserArchiveListSourceFavorite); got < 0 {
		t.Fatalf("sync archive should return elapsed ms, got %d", got)
	}
	if stub.persistCalls != 1 || stub.lastOwner != "me" || stub.lastSource != UserArchiveListSourceFavorite {
		t.Fatalf("stub call unexpected: %+v", stub)
	}

	if copied := cloneUserListForArchive(nil); copied != nil {
		t.Fatalf("clone nil should return nil")
	}
	orig := []map[string]any{nil, {"id": "u2", "name": "before"}}
	cloned := cloneUserListForArchive(orig)
	if len(cloned) != 1 || toString(cloned[0]["id"]) != "u2" {
		t.Fatalf("clone result unexpected: %+v", cloned)
	}
	orig[1]["name"] = "after"
	if toString(cloned[0]["name"]) != "before" {
		t.Fatalf("clone should be deep copy: %+v", cloned[0])
	}
}

func TestPersistArchivedUserList_DBArchiveAsyncBranch(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
		WithArgs("me", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "seen_in_history", "seen_in_favorite", "snapshot_json", "last_msg", "last_time"}))
	mock.ExpectExec(`INSERT INTO chat_user_archive`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
	app := &App{userArchive: svc}
	if got := app.persistArchivedUserList("me", []map[string]any{{"id": "u1", "lastMsg": "x"}}, UserArchiveListSourceHistory); got < 0 {
		t.Fatalf("async archive should return elapsed ms, got %d", got)
	}

	deadline := time.Now().Add(800 * time.Millisecond)
	for {
		if err := mock.ExpectationsWereMet(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("async archive did not finish in time")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
