package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserArchiveHelpers(t *testing.T) {
	t.Run("source flags and filter column", func(t *testing.T) {
		h, f := sourceFlags(UserArchiveListSourceHistory)
		if h != 1 || f != 0 {
			t.Fatalf("history flags=(%d,%d)", h, f)
		}
		h, f = sourceFlags(UserArchiveListSourceFavorite)
		if h != 0 || f != 1 {
			t.Fatalf("favorite flags=(%d,%d)", h, f)
		}
		if got := sourceFilterColumn(UserArchiveListSourceFavorite); got != "seen_in_favorite" {
			t.Fatalf("column=%q", got)
		}
		if got := sourceFilterColumn(UserArchiveListSourceHistory); got != "seen_in_history" {
			t.Fatalf("column=%q", got)
		}
	})

	t.Run("sanitize snapshot removes localArchived and fills id", func(t *testing.T) {
		raw := map[string]any{
			"UserID":        "u2",
			"nickname":      "Bob",
			"localArchived": true,
			" ":             "ignored",
		}
		out := sanitizeArchivedUserSnapshot(raw, "fallback")
		if _, ok := out["localArchived"]; ok {
			t.Fatalf("localArchived should be removed: %v", out)
		}
		if got := toString(out["id"]); got != "u2" {
			t.Fatalf("id=%q, want u2", got)
		}
		if got := toString(out["nickname"]); got != "Bob" {
			t.Fatalf("nickname=%q", got)
		}
	})

	t.Run("nullable string trims empty", func(t *testing.T) {
		if got := nullableString("   "); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
		if got := nullableString(" x "); got != "x" {
			t.Fatalf("expected x, got %v", got)
		}
	})
}

func TestDBUserArchiveService_PersistUserList_FavoriteInsert(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
		WithArgs("me", "u2").
		WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "seen_in_history", "seen_in_favorite", "snapshot_json", "last_msg", "last_time"}))

	mock.ExpectExec(`INSERT INTO chat_user_archive`).
		WithArgs(
			"me",
			"u2",
			sqlmock.AnyArg(),
			"hello",
			"2026-02-27 12:00:00",
			0,
			1,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
	users := []map[string]any{{
		"id":       "u2",
		"nickname": "Bob",
		"lastMsg":  "hello",
		"lastTime": "2026-02-27 12:00:00",
	}}
	svc.PersistUserList(context.Background(), "me", users, UserArchiveListSourceFavorite)
}

func TestDBUserArchiveService_PersistUserList_SkipUnchanged(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	defer cleanup()

	snapshot, err := json.Marshal(map[string]any{
		"id":       "u2",
		"nickname": "Bob",
		"lastMsg":  "hello",
		"lastTime": "2026-02-27 12:00:00",
	})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
		WithArgs("me", "u2").
		WillReturnRows(
			sqlmock.NewRows([]string{"target_user_id", "seen_in_history", "seen_in_favorite", "snapshot_json", "last_msg", "last_time"}).
				AddRow("u2", 0, 1, string(snapshot), "hello", "2026-02-27 12:00:00"),
		)

	svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
	users := []map[string]any{{
		"id":       "u2",
		"nickname": "Bob",
		"lastMsg":  "hello",
		"lastTime": "2026-02-27 12:00:00",
	}}
	svc.PersistUserList(context.Background(), "me", users, UserArchiveListSourceFavorite)
}

func TestDBUserArchiveService_MergeArchivedUsers_AppendsLocalArchived(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	defer cleanup()

	snapshot, err := json.Marshal(map[string]any{"nickname": "ArchivedUser"})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	mock.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND seen_in_history = 1\s+ORDER BY updated_at DESC, id DESC`).
		WithArgs("me").
		WillReturnRows(
			sqlmock.NewRows([]string{"target_user_id", "snapshot_json", "last_msg", "last_time"}).
				AddRow("u2", sql.NullString{String: string(snapshot), Valid: true}, sql.NullString{String: "msg2", Valid: true}, sql.NullString{String: "t2", Valid: true}),
		)

	svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
	upstream := []map[string]any{{"id": "u1", "nickname": "UpstreamUser"}}
	merged := svc.MergeArchivedUsers(context.Background(), "me", upstream, UserArchiveListSourceHistory)

	if len(merged) != 2 {
		t.Fatalf("len=%d merged=%v", len(merged), merged)
	}
	if got := toString(merged[1]["id"]); got != "u2" {
		t.Fatalf("id=%q", got)
	}
	if got := toString(merged[1]["lastMsg"]); got != "msg2" {
		t.Fatalf("lastMsg=%q", got)
	}
	if got := toString(merged[1]["lastTime"]); got != "t2" {
		t.Fatalf("lastTime=%q", got)
	}
	if archived, ok := merged[1]["localArchived"].(bool); !ok || !archived {
		t.Fatalf("localArchived=%v", merged[1]["localArchived"])
	}
}

func TestDBUserArchiveService_ListContactCandidates(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	defer cleanup()

	snapshot, err := json.Marshal(map[string]any{
		"id":         "u2",
		"nickname":   "Bob",
		"cookieData": "secret",
	})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	mock.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time, seen_in_history, seen_in_favorite\s+FROM chat_user_archive\s+WHERE owner_user_id = \?\s+ORDER BY last_time DESC, last_seen_at DESC, updated_at DESC, id DESC\s+LIMIT \?`).
		WithArgs("me", 25).
		WillReturnRows(
			sqlmock.NewRows([]string{"target_user_id", "snapshot_json", "last_msg", "last_time", "seen_in_history", "seen_in_favorite"}).
				AddRow("u2", sql.NullString{String: string(snapshot), Valid: true}, sql.NullString{String: "hi", Valid: true}, sql.NullString{String: "t1", Valid: true}, 1, 0),
		)

	svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
	items, err := svc.ListContactCandidates(context.Background(), " me ", 25)
	if err != nil {
		t.Fatalf("ListContactCandidates: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%+v", items)
	}
	if items[0].TargetUserID != "u2" || items[0].Nickname != "Bob" || !items[0].LocalArchived {
		t.Fatalf("item=%+v", items[0])
	}
	if items[0].LastMsg != "hi" || items[0].LastTime != "t1" {
		t.Fatalf("last fields=%+v", items[0])
	}
	if _, ok := items[0].Snapshot["cookieData"]; ok {
		t.Fatalf("sensitive snapshot leaked: %v", items[0].Snapshot)
	}
}

func TestDBUserArchiveService_SearchArchive(t *testing.T) {
	const expectArchiveSearchSQL = `SELECT owner_user_id, target_user_id, snapshot_json, last_msg, last_time, seen_in_history, seen_in_favorite\s+FROM chat_user_archive\s+WHERE LOWER\(target_user_id\) LIKE \? OR LOWER\(COALESCE\(snapshot_json, ''\)\) LIKE \? OR LOWER\(COALESCE\(last_msg, ''\)\) LIKE \?\s+ORDER BY last_seen_at DESC, updated_at DESC, id DESC\s+LIMIT \?`

	t.Run("matches target_user_id like and strips sensitive snapshot fields", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		snapshot, err := json.Marshal(map[string]any{
			"id":         "target-abc",
			"nickname":   "Alice Target",
			"cookieData": "secret",
		})
		if err != nil {
			t.Fatalf("marshal snapshot: %v", err)
		}

		mock.ExpectQuery(expectArchiveSearchSQL).
			WithArgs("%target%", "%target%", "%target%", 20).
			WillReturnRows(
				sqlmock.NewRows([]string{"owner_user_id", "target_user_id", "snapshot_json", "last_msg", "last_time", "seen_in_history", "seen_in_favorite"}).
					AddRow("owner-a", "target-abc", sql.NullString{String: string(snapshot), Valid: true}, sql.NullString{String: "hi", Valid: true}, sql.NullString{String: "t1", Valid: true}, 1, 0),
			)

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		items, err := svc.SearchArchive(context.Background(), " Target ", 20)
		if err != nil {
			t.Fatalf("SearchArchive: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("items=%+v", items)
		}
		if items[0].OwnerUserID != "owner-a" || items[0].TargetUserID != "target-abc" || items[0].Nickname != "Alice Target" {
			t.Fatalf("item=%+v", items[0])
		}
		if _, ok := items[0].Snapshot["cookieData"]; ok {
			t.Fatalf("sensitive snapshot leaked: %v", items[0].Snapshot)
		}
	})

	t.Run("matches parsed snapshot name fields without owner filter", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		snapshot, err := json.Marshal(map[string]any{
			"id":             "u2",
			"targetUserName": "Snapshot Name",
		})
		if err != nil {
			t.Fatalf("marshal snapshot: %v", err)
		}

		mock.ExpectQuery(expectArchiveSearchSQL).
			WithArgs("%snapshot%", "%snapshot%", "%snapshot%", 50).
			WillReturnRows(
				sqlmock.NewRows([]string{"owner_user_id", "target_user_id", "snapshot_json", "last_msg", "last_time", "seen_in_history", "seen_in_favorite"}).
					AddRow("owner-a", "u2", sql.NullString{String: string(snapshot), Valid: true}, sql.NullString{}, sql.NullString{}, 0, 1),
			)

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		items, err := svc.SearchArchive(context.Background(), "snapshot", 50)
		if err != nil {
			t.Fatalf("SearchArchive: %v", err)
		}
		if len(items) != 1 || items[0].TargetUserName != "Snapshot Name" || items[0].OwnerUserID != "owner-a" {
			t.Fatalf("items=%+v", items)
		}
		if !containsString(items[0].Sources, "favorite") {
			t.Fatalf("sources=%v", items[0].Sources)
		}
	})

	t.Run("returns cross identity results", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		snapshotA, _ := json.Marshal(map[string]any{"id": "u1", "nickname": "Target One"})
		snapshotB, _ := json.Marshal(map[string]any{"id": "u2", "nickname": "Target Two"})
		mock.ExpectQuery(expectArchiveSearchSQL).
			WithArgs("%target%", "%target%", "%target%", 100).
			WillReturnRows(
				sqlmock.NewRows([]string{"owner_user_id", "target_user_id", "snapshot_json", "last_msg", "last_time", "seen_in_history", "seen_in_favorite"}).
					AddRow("owner-a", "u1", sql.NullString{String: string(snapshotA), Valid: true}, sql.NullString{}, sql.NullString{}, 1, 0).
					AddRow("owner-b", "u2", sql.NullString{String: string(snapshotB), Valid: true}, sql.NullString{}, sql.NullString{}, 1, 1),
			)

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		items, err := svc.SearchArchive(context.Background(), "target", 100)
		if err != nil {
			t.Fatalf("SearchArchive: %v", err)
		}
		if len(items) != 2 || items[0].OwnerUserID != "owner-a" || items[1].OwnerUserID != "owner-b" {
			t.Fatalf("items=%+v", items)
		}
	})

	t.Run("bad json does not abort target id match", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(expectArchiveSearchSQL).
			WithArgs("%target%", "%target%", "%target%", 100).
			WillReturnRows(
				sqlmock.NewRows([]string{"owner_user_id", "target_user_id", "snapshot_json", "last_msg", "last_time", "seen_in_history", "seen_in_favorite"}).
					AddRow("owner-a", "target-bad", sql.NullString{String: "{", Valid: true}, sql.NullString{String: "hi", Valid: true}, sql.NullString{String: "t1", Valid: true}, 1, 0),
			)

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		items, err := svc.SearchArchive(context.Background(), "target", 100)
		if err != nil {
			t.Fatalf("SearchArchive: %v", err)
		}
		if len(items) != 1 || items[0].TargetUserID != "target-bad" {
			t.Fatalf("items=%+v", items)
		}
	})

	t.Run("matches last message field", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		snapshot, err := json.Marshal(map[string]any{
			"id":       "u2",
			"nickname": "Bob",
		})
		if err != nil {
			t.Fatalf("marshal snapshot: %v", err)
		}

		mock.ExpectQuery(expectArchiveSearchSQL).
			WithArgs("%hello%", "%hello%", "%hello%", 100).
			WillReturnRows(
				sqlmock.NewRows([]string{"owner_user_id", "target_user_id", "snapshot_json", "last_msg", "last_time", "seen_in_history", "seen_in_favorite"}).
					AddRow("owner-a", "u2", sql.NullString{String: string(snapshot), Valid: true}, sql.NullString{String: "hello from archive", Valid: true}, sql.NullString{}, 1, 0),
			)

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		items, err := svc.SearchArchive(context.Background(), "hello", 100)
		if err != nil {
			t.Fatalf("SearchArchive: %v", err)
		}
		if len(items) != 1 || items[0].LastMsg != "hello from archive" {
			t.Fatalf("items=%+v", items)
		}
	})

	t.Run("raw sensitive snapshot hit is rejected by parsed field filter", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		snapshot, err := json.Marshal(map[string]any{
			"id":         "u2",
			"nickname":   "Bob",
			"cookieData": "secret-value",
		})
		if err != nil {
			t.Fatalf("marshal snapshot: %v", err)
		}

		mock.ExpectQuery(expectArchiveSearchSQL).
			WithArgs("%secret%", "%secret%", "%secret%", 100).
			WillReturnRows(
				sqlmock.NewRows([]string{"owner_user_id", "target_user_id", "snapshot_json", "last_msg", "last_time", "seen_in_history", "seen_in_favorite"}).
					AddRow("owner-a", "u2", sql.NullString{String: string(snapshot), Valid: true}, sql.NullString{}, sql.NullString{}, 1, 0),
			)

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		items, err := svc.SearchArchive(context.Background(), "secret", 100)
		if err != nil {
			t.Fatalf("SearchArchive: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("items=%+v", items)
		}
	})
}

func TestDBUserArchiveService_DeleteConversation(t *testing.T) {
	t.Run("delete by owner and target", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`DELETE FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \?`).
			WithArgs("me", "u2").
			WillReturnResult(sqlmock.NewResult(0, 1))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		svc.DeleteConversation(context.Background(), " me ", " u2 ")
	})

	t.Run("noop on empty params", func(t *testing.T) {
		rawDB, _, cleanup := newSQLMock(t)
		defer cleanup()

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		svc.DeleteConversation(context.Background(), " ", "u2")
		svc.DeleteConversation(context.Background(), "me", " ")
	})
}
