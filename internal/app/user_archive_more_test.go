package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserArchiveHelperBranches(t *testing.T) {
	if NewDBUserArchiveService(nil) != nil {
		t.Fatalf("nil db should return nil service")
	}

	user := map[string]any{"id": "keep"}
	ensureArchivedUserID(user, "fallback")
	if user["id"] != "keep" {
		t.Fatalf("id should keep existing value")
	}

	cases := []struct {
		name string
		in   map[string]any
		want string
	}{
		{name: "UserID", in: map[string]any{"UserID": "u1"}, want: "u1"},
		{name: "userid", in: map[string]any{"userid": "u2"}, want: "u2"},
		{name: "userId", in: map[string]any{"userId": "u3"}, want: "u3"},
		{name: "fallback", in: map[string]any{"name": "x"}, want: "u4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ensureArchivedUserID(tc.in, "u4")
			if got := toString(tc.in["id"]); got != tc.want {
				t.Fatalf("id=%q, want %q", got, tc.want)
			}
		})
	}

	incoming := archiveUpsertInput{TargetUserID: "u", LastMsg: "m", SeenInHistory: 1, SeenAt: time.Now().Add(time.Second)}
	if got := mergeArchiveUpsertInput(archiveUpsertInput{}, incoming); got.TargetUserID != "u" {
		t.Fatalf("expected incoming when base target empty")
	}

	base := archiveUpsertInput{TargetUserID: "u", SnapshotJSON: "a", LastMsg: "", LastTime: "", SeenAt: time.Now()}
	merged := mergeArchiveUpsertInput(base, archiveUpsertInput{TargetUserID: "u", LastMsg: "msg", LastTime: "2026", SeenInFavorite: 1, SeenAt: time.Now().Add(2 * time.Second)})
	if merged.LastMsg != "msg" || merged.LastTime != "2026" || merged.SeenInFavorite != 1 {
		t.Fatalf("merge result unexpected: %+v", merged)
	}

	if mergeSeenFlag(1, 0) != 1 || mergeSeenFlag(0, 1) != 1 || mergeSeenFlag(0, 0) != 0 {
		t.Fatalf("mergeSeenFlag unexpected")
	}
	if archiveFieldChanged("", "x") {
		t.Fatalf("empty incoming should not be changed")
	}
	if !archiveFieldChanged(" x ", "y") {
		t.Fatalf("different field should be changed")
	}
	if archiveRowNeedsUpdate(
		archiveUpsertInput{SeenInHistory: 1, SeenInFavorite: 0, SnapshotJSON: "a", LastMsg: "m", LastTime: "t"},
		archiveExistingState{SeenInHistory: 1, SeenInFavorite: 0, SnapshotJSON: "a", LastMsg: "m", LastTime: "t"},
	) {
		t.Fatalf("identical row should not require update")
	}
	if !archiveRowNeedsUpdate(
		archiveUpsertInput{SeenInHistory: 1, SeenInFavorite: 1},
		archiveExistingState{SeenInHistory: 1, SeenInFavorite: 0},
	) {
		t.Fatalf("seen flag changed should require update")
	}
}

func TestDBUserArchiveService_PersistUserList_MarshalAndQueryError(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
		WithArgs("me", "u2").
		WillReturnError(errors.New("query failed"))

	svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
	svc.PersistUserList(nil, " me ", []map[string]any{
		{"id": "u1", "bad": make(chan int)},
		{"id": "u2", "lastMsg": "hi", "lastTime": "2026-03-01 00:00:00"},
	}, UserArchiveListSourceHistory)
}

func TestDBUserArchiveService_MergeArchivedUsers_Branches(t *testing.T) {
	t.Run("owner empty or service nil", func(t *testing.T) {
		upstream := []map[string]any{{"id": "u1"}}
		svc := &DBUserArchiveService{}
		if got := svc.MergeArchivedUsers(context.Background(), " ", upstream, UserArchiveListSourceHistory); len(got) != 1 {
			t.Fatalf("owner empty should return upstream")
		}
		var nilSvc *DBUserArchiveService
		if got := nilSvc.MergeArchivedUsers(context.Background(), "me", upstream, UserArchiveListSourceHistory); len(got) != 1 {
			t.Fatalf("nil service should return upstream")
		}
	})

	t.Run("query error", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND seen_in_history = 1\s+ORDER BY updated_at DESC, id DESC`).
			WithArgs("me").
			WillReturnError(errors.New("boom"))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		upstream := []map[string]any{{"id": "u1"}}
		if got := svc.MergeArchivedUsers(context.Background(), "me", upstream, UserArchiveListSourceHistory); len(got) != 1 {
			t.Fatalf("query error should return upstream")
		}
	})

	t.Run("invalid snapshot and merge defaults", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{"target_user_id", "snapshot_json", "last_msg", "last_time"}).
			AddRow("u1", sql.NullString{String: `{"id":"u1"}`, Valid: true}, sql.NullString{String: "ignored", Valid: true}, sql.NullString{String: "ignored", Valid: true}).
			AddRow(" ", sql.NullString{String: "{}", Valid: true}, sql.NullString{}, sql.NullString{}).
			AddRow("u2", sql.NullString{String: "{", Valid: true}, sql.NullString{String: "last2", Valid: true}, sql.NullString{String: "t2", Valid: true}).
			AddRow("u3", sql.NullString{String: "", Valid: false}, sql.NullString{String: "last3", Valid: true}, sql.NullString{String: "t3", Valid: true})

		mock.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND seen_in_history = 1\s+ORDER BY updated_at DESC, id DESC`).
			WithArgs("me").
			WillReturnRows(rows)

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		upstream := []map[string]any{nil, {"id": "u1", "nickname": "up"}}
		merged := svc.MergeArchivedUsers(context.Background(), "me", upstream, UserArchiveListSourceHistory)
		if len(merged) != 3 {
			t.Fatalf("len=%d, want 3", len(merged))
		}
		if toString(merged[1]["id"]) != "u2" || toString(merged[1]["lastMsg"]) != "last2" || toString(merged[1]["lastTime"]) != "t2" {
			t.Fatalf("u2 merged unexpected: %+v", merged[1])
		}
		if merged[1]["localArchived"] != true {
			t.Fatalf("u2 localArchived should be true")
		}
		if toString(merged[2]["id"]) != "u3" || toString(merged[2]["lastMsg"]) != "last3" {
			t.Fatalf("u3 merged unexpected: %+v", merged[2])
		}
	})
}

func TestDBUserArchiveService_TouchSaveDeleteBranches(t *testing.T) {
	t.Run("touch inserts when missing", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"seen_in_history", "seen_in_favorite"}))
		mock.ExpectExec(`INSERT INTO chat_user_archive`).
			WithArgs("me", "u1", nil, nil, nil, 1, 0, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		svc.TouchConversation(nil, " me ", " u1 ")
	})

	t.Run("save last message noop on empty content and time", func(t *testing.T) {
		rawDB, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		svc.SaveLastMessage(context.Background(), "me", "u1", " ", " ")
	})

	t.Run("save handles duplicate insert by update", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u2").
			WillReturnRows(sqlmock.NewRows([]string{"seen_in_history", "seen_in_favorite"}))
		mock.ExpectExec(`INSERT INTO chat_user_archive`).
			WithArgs("me", "u2", nil, "hello", "2026-03-01", 1, 0, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(duplicateKeyErr())
		mock.ExpectExec(`UPDATE chat_user_archive`).
			WithArgs(nil, "hello", "2026-03-01", 1, 0, sqlmock.AnyArg(), sqlmock.AnyArg(), "me", "u2").
			WillReturnResult(sqlmock.NewResult(0, 1))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		svc.SaveLastMessage(context.Background(), "me", "u2", "hello", "2026-03-01")
	})

	t.Run("save updates existing row", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u3").
			WillReturnRows(sqlmock.NewRows([]string{"seen_in_history", "seen_in_favorite"}).AddRow(0, 1))
		mock.ExpectExec(`UPDATE chat_user_archive`).
			WithArgs(nil, "hello", "2026-03-02", 1, 1, sqlmock.AnyArg(), sqlmock.AnyArg(), "me", "u3").
			WillReturnResult(sqlmock.NewResult(0, 1))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		svc.SaveLastMessage(context.Background(), "me", "u3", "hello", "2026-03-02")
	})

	t.Run("delete swallows exec error", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`DELETE FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \?`).
			WithArgs("me", "u4").
			WillReturnError(errors.New("delete failed"))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		svc.DeleteConversation(context.Background(), "me", "u4")
	})
}

func TestDBUserArchiveService_UpsertRowsForList_Branches(t *testing.T) {
	t.Run("owner mismatch", func(t *testing.T) {
		rawDB, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))

		_, _, err := svc.upsertRowsForList(context.Background(), []archiveUpsertInput{
			{OwnerUserID: "a", TargetUserID: "u1", SeenAt: time.Now()},
			{OwnerUserID: "b", TargetUserID: "u2", SeenAt: time.Now()},
		})
		if err == nil {
			t.Fatalf("expected owner mismatch error")
		}
	})

	t.Run("dedup skip unchanged and write delta", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?,\?\)`).
			WithArgs("me", "u1", "u2").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "seen_in_history", "seen_in_favorite", "snapshot_json", "last_msg", "last_time"}).
				AddRow("u1", 1, 0, "s1", "m1", "t1"))
		mock.ExpectExec(`INSERT INTO chat_user_archive`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		written, skipped, err := svc.upsertRowsForList(context.Background(), []archiveUpsertInput{
			{OwnerUserID: "me", TargetUserID: "u1", SnapshotJSON: "s1", LastMsg: "m1", LastTime: "t1", SeenInHistory: 1, SeenAt: time.Now()},
			{OwnerUserID: "me", TargetUserID: "u1", SeenInFavorite: 1, SeenAt: time.Now().Add(time.Second)},
			{OwnerUserID: "me", TargetUserID: "u2", SnapshotJSON: "s2", LastMsg: "m2", LastTime: "t2", SeenInHistory: 1, SeenAt: time.Now()},
		})
		if err != nil {
			t.Fatalf("upsertRowsForList error: %v", err)
		}
		if written != 2 || skipped != 0 {
			t.Fatalf("written=%d skipped=%d", written, skipped)
		}
	})

	t.Run("all skipped when unchanged", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
			WithArgs("me", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "seen_in_history", "seen_in_favorite", "snapshot_json", "last_msg", "last_time"}).
				AddRow("u1", 1, 0, "s1", "m1", "t1"))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		written, skipped, err := svc.upsertRowsForList(context.Background(), []archiveUpsertInput{{OwnerUserID: "me", TargetUserID: "u1", SnapshotJSON: "s1", LastMsg: "m1", LastTime: "t1", SeenInHistory: 1, SeenAt: time.Now()}})
		if err != nil {
			t.Fatalf("upsertRowsForList error: %v", err)
		}
		if written != 0 || skipped != 1 {
			t.Fatalf("written=%d skipped=%d", written, skipped)
		}
	})

	t.Run("batch write error", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
			WithArgs("me", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "seen_in_history", "seen_in_favorite", "snapshot_json", "last_msg", "last_time"}))
		mock.ExpectExec(`INSERT INTO chat_user_archive`).
			WillReturnError(errors.New("write failed"))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		if _, _, err := svc.upsertRowsForList(context.Background(), []archiveUpsertInput{{OwnerUserID: "me", TargetUserID: "u1", SnapshotJSON: "s1", SeenInHistory: 1, SeenAt: time.Now()}}); err == nil {
			t.Fatalf("expected write failed")
		}
	})
}

func TestDBUserArchiveService_QueryHelpersAndPostgresBatch(t *testing.T) {
	t.Run("fetchExistingStates query/scan errors", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))

		mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
			WithArgs("me", "u1").
			WillReturnError(errors.New("query err"))
		if _, err := svc.fetchExistingStates(context.Background(), "me", []string{"u1"}); err == nil {
			t.Fatalf("expected query err")
		}

		rawDB2, mock2, cleanup2 := newSQLMock(t)
		defer cleanup2()
		svc2 := NewDBUserArchiveService(wrapMySQLDB(rawDB2))
		mock2.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
			WithArgs("me", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "seen_in_history", "seen_in_favorite", "snapshot_json", "last_msg", "last_time"}).
				AddRow("u1", "bad-int", 0, "", "", ""))
		if _, err := svc2.fetchExistingStates(context.Background(), "me", []string{"u1"}); err == nil {
			t.Fatalf("expected scan err")
		}
	})

	t.Run("fetchSeenFlags and listArchivedRows branches", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"seen_in_history", "seen_in_favorite"}))
		if _, _, exists, err := svc.fetchSeenFlags(context.Background(), "me", "u1"); err != nil || exists {
			t.Fatalf("no rows should not exist: exists=%v err=%v", exists, err)
		}

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u2").
			WillReturnError(errors.New("scan err"))
		if _, _, _, err := svc.fetchSeenFlags(context.Background(), "me", "u2"); err == nil {
			t.Fatalf("expected scan err")
		}

		mock.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND seen_in_favorite = 1\s+ORDER BY updated_at DESC, id DESC`).
			WithArgs("me").
			WillReturnError(errors.New("list query err"))
		if _, err := svc.listArchivedRows(context.Background(), "me", UserArchiveListSourceFavorite); err == nil {
			t.Fatalf("expected list query err")
		}

		mock.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND seen_in_history = 1\s+ORDER BY updated_at DESC, id DESC`).
			WithArgs("me").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "snapshot_json", "last_msg", "last_time"}).
				AddRow("u3", sql.NullString{}, sql.NullString{}, sql.NullString{}).
				RowError(0, errors.New("row scan failed")))
		if _, err := svc.listArchivedRows(context.Background(), "me", UserArchiveListSourceHistory); err == nil {
			t.Fatalf("expected scan err")
		}
	})

	t.Run("upsertRowsBatch postgres branch", func(t *testing.T) {
		t.Setenv("TEST_DB_DIALECT", "postgres")
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		mock.ExpectExec(`ON CONFLICT \(owner_user_id, target_user_id\) DO UPDATE`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := svc.upsertRowsBatch(context.Background(), []archiveUpsertInput{{
			OwnerUserID:   "me",
			TargetUserID:  "u1",
			SnapshotJSON:  "{}",
			LastMsg:       "m",
			LastTime:      "t",
			SeenInHistory: 1,
			SeenAt:        time.Now(),
		}})
		if err != nil {
			t.Fatalf("upsertRowsBatch postgres err: %v", err)
		}
	})
}

func TestDBUserArchiveService_AdditionalEdgeCoverage(t *testing.T) {
	t.Run("persist short-circuit and prepared empty", func(t *testing.T) {
		var nilSvc *DBUserArchiveService
		nilSvc.PersistUserList(nil, "me", []map[string]any{{"id": "u1"}}, UserArchiveListSourceHistory)

		rawDB, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		svc.PersistUserList(nil, "me", []map[string]any{{"id": "  "}, {"name": "x"}}, UserArchiveListSourceHistory)
	})

	t.Run("merge archived users empty rows", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND seen_in_favorite = 1\s+ORDER BY updated_at DESC, id DESC`).
			WithArgs("me").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "snapshot_json", "last_msg", "last_time"}))
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		upstream := []map[string]any{{"id": "u1"}}
		if got := svc.MergeArchivedUsers(nil, "me", upstream, UserArchiveListSourceFavorite); len(got) != 1 {
			t.Fatalf("expected upstream when archived rows empty")
		}
	})

	t.Run("touch/save/delete error branches", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u1").
			WillReturnError(errors.New("query failed"))
		svc.TouchConversation(nil, "me", "u1")

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u2").
			WillReturnRows(sqlmock.NewRows([]string{"seen_in_history", "seen_in_favorite"}))
		mock.ExpectExec(`INSERT INTO chat_user_archive`).
			WithArgs("me", "u2", nil, "msg", "2026", 1, 0, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert failed"))
		svc.SaveLastMessage(nil, "me", "u2", "msg", "2026")

		mock.ExpectExec(`DELETE FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \?`).
			WithArgs("me", "u3").
			WillReturnResult(sqlmock.NewResult(0, 1))
		svc.DeleteConversation(nil, "me", "u3")
		svc.DeleteConversation(nil, " ", "u3")
	})

	t.Run("upsertRowsForList skipped invalid rows", func(t *testing.T) {
		rawDB, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		written, skipped, err := svc.upsertRowsForList(nil, []archiveUpsertInput{{OwnerUserID: "", TargetUserID: "u1"}, {OwnerUserID: "me", TargetUserID: ""}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if written != 0 || skipped != 2 {
			t.Fatalf("written=%d skipped=%d", written, skipped)
		}
	})

	t.Run("fetchExistingStates rows err", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))

		mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
			WithArgs("me", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "seen_in_history", "seen_in_favorite", "snapshot_json", "last_msg", "last_time"}).
				AddRow("u1", 1, 0, "s", "m", "t").
				RowError(0, errors.New("rows err")))

		if _, err := svc.fetchExistingStates(nil, "me", []string{"u1"}); err == nil {
			t.Fatalf("expected rows err")
		}

		if result, err := (*DBUserArchiveService)(nil).fetchExistingStates(nil, "", nil); err != nil || len(result) != 0 {
			t.Fatalf("nil svc fetchExistingStates should return empty result")
		}
	})

	t.Run("upsertRowsBatch nil and zero time branches", func(t *testing.T) {
		if err := (*DBUserArchiveService)(nil).upsertRowsBatch(nil, nil); err != nil {
			t.Fatalf("nil svc upsertRowsBatch should be nil err")
		}

		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		mock.ExpectExec(`ON DUPLICATE KEY UPDATE`).WillReturnResult(sqlmock.NewResult(1, 1))
		if err := svc.upsertRowsBatch(nil, []archiveUpsertInput{{OwnerUserID: "me", TargetUserID: "u1", SeenInHistory: 1}}); err != nil {
			t.Fatalf("upsertRowsBatch err: %v", err)
		}
	})

	t.Run("upsertRow early return and update error", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))

		if err := svc.upsertRow(nil, archiveUpsertInput{}); err != nil {
			t.Fatalf("empty owner/target should return nil")
		}

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"seen_in_history", "seen_in_favorite"}).AddRow(0, 0))
		mock.ExpectExec(`UPDATE chat_user_archive`).
			WillReturnError(errors.New("update failed"))
		if err := svc.upsertRow(nil, archiveUpsertInput{OwnerUserID: "me", TargetUserID: "u1", SeenInHistory: 1}); err == nil {
			t.Fatalf("expected update error")
		}
	})

	t.Run("fetchSeenFlags ctx nil success and listArchivedRows row scan", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"seen_in_history", "seen_in_favorite"}).AddRow(1, 1))
		h, f, exists, err := svc.fetchSeenFlags(nil, "me", "u1")
		if err != nil || !exists || h != 1 || f != 1 {
			t.Fatalf("fetchSeenFlags got h=%d f=%d exists=%v err=%v", h, f, exists, err)
		}

		mock.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND seen_in_history = 1\s+ORDER BY updated_at DESC, id DESC`).
			WithArgs("me").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "snapshot_json", "last_msg", "last_time"}).
				AddRow("u2", sql.NullString{String: "{\"id\":\"u2\"}", Valid: true}, sql.NullString{String: "m", Valid: true}, sql.NullString{String: "t", Valid: true}))
		rows, err := svc.listArchivedRows(nil, "me", UserArchiveListSourceHistory)
		if err != nil || len(rows) != 1 || rows[0].TargetUserID != "u2" {
			t.Fatalf("listArchivedRows got rows=%+v err=%v", rows, err)
		}
	})
}

func TestDBUserArchiveService_UncoveredBranchCoverage(t *testing.T) {
	t.Run("merge snapshot null and helper nil inputs", func(t *testing.T) {
		if got := sanitizeArchivedUserSnapshot(nil, "u1"); toString(got["id"]) != "u1" {
			t.Fatalf("sanitize nil user got=%v", got)
		}
		ensureArchivedUserID(nil, "u1")

		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND seen_in_history = 1\s+ORDER BY updated_at DESC, id DESC`).
			WithArgs("me").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "snapshot_json", "last_msg", "last_time"}).
				AddRow("u2", sql.NullString{String: "null", Valid: true}, sql.NullString{String: "m2", Valid: true}, sql.NullString{String: "t2", Valid: true}))

		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))
		out := svc.MergeArchivedUsers(context.Background(), "me", []map[string]any{{"id": "u1"}}, UserArchiveListSourceHistory)
		if len(out) != 2 || toString(out[1]["id"]) != "u2" || toString(out[1]["lastMsg"]) != "m2" {
			t.Fatalf("unexpected merge result: %v", out)
		}
	})

	t.Run("merge/input helpers and row update fields", func(t *testing.T) {
		merged := mergeArchiveUpsertInput(
			archiveUpsertInput{TargetUserID: "u1", SnapshotJSON: "old", SeenInHistory: 0},
			archiveUpsertInput{TargetUserID: "u1", SnapshotJSON: "new", SeenInHistory: 1},
		)
		if merged.SnapshotJSON != "new" || merged.SeenInHistory != 1 {
			t.Fatalf("merged=%+v", merged)
		}

		base := archiveExistingState{SeenInHistory: 1, SeenInFavorite: 0, SnapshotJSON: "s", LastMsg: "m", LastTime: "t"}
		if !archiveRowNeedsUpdate(archiveUpsertInput{SeenInHistory: 0, SeenInFavorite: 0}, base) {
			t.Fatalf("seen history diff should require update")
		}
		if !archiveRowNeedsUpdate(archiveUpsertInput{SeenInHistory: 1, SeenInFavorite: 0, SnapshotJSON: "x"}, base) {
			t.Fatalf("snapshot diff should require update")
		}
		if !archiveRowNeedsUpdate(archiveUpsertInput{SeenInHistory: 1, SeenInFavorite: 0, LastMsg: "x"}, base) {
			t.Fatalf("lastMsg diff should require update")
		}
		if !archiveRowNeedsUpdate(archiveUpsertInput{SeenInHistory: 1, SeenInFavorite: 0, LastTime: "x"}, base) {
			t.Fatalf("lastTime diff should require update")
		}
	})

	t.Run("touch/save nil receiver and upsertRowsForList short-circuit", func(t *testing.T) {
		var nilSvc *DBUserArchiveService
		nilSvc.TouchConversation(nil, "me", "u1")
		nilSvc.SaveLastMessage(nil, "me", "u1", "m", "t")
		if written, skipped, err := nilSvc.upsertRowsForList(nil, nil); err != nil || written != 0 || skipped != 0 {
			t.Fatalf("nil upsertRowsForList got written=%d skipped=%d err=%v", written, skipped, err)
		}
	})

	t.Run("upsertRowsForList zero SeenAt and listArchivedRows scan error", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))

		mock.ExpectQuery(`SELECT target_user_id, seen_in_history, seen_in_favorite, COALESCE\(snapshot_json, ''\), COALESCE\(last_msg, ''\), COALESCE\(last_time, ''\)\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND target_user_id IN \(\?\)`).
			WithArgs("me", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "seen_in_history", "seen_in_favorite", "snapshot_json", "last_msg", "last_time"}))
		mock.ExpectExec(`INSERT INTO chat_user_archive`).WillReturnResult(sqlmock.NewResult(1, 1))

		written, skipped, err := svc.upsertRowsForList(context.Background(), []archiveUpsertInput{{OwnerUserID: "me", TargetUserID: "u1", SeenInHistory: 1}})
		if err != nil || written != 1 || skipped != 0 {
			t.Fatalf("written=%d skipped=%d err=%v", written, skipped, err)
		}

		rawDB2, mock2, cleanup2 := newSQLMock(t)
		defer cleanup2()
		svc2 := NewDBUserArchiveService(wrapMySQLDB(rawDB2))
		mock2.ExpectQuery(`SELECT target_user_id, snapshot_json, last_msg, last_time\s+FROM chat_user_archive\s+WHERE owner_user_id = \? AND seen_in_history = 1\s+ORDER BY updated_at DESC, id DESC`).
			WithArgs("me").
			WillReturnRows(sqlmock.NewRows([]string{"target_user_id", "snapshot_json", "last_msg", "last_time"}).
				AddRow(nil, sql.NullString{}, sql.NullString{}, sql.NullString{}))
		if _, err := svc2.listArchivedRows(context.Background(), "me", UserArchiveListSourceHistory); err == nil {
			t.Fatalf("expected listArchivedRows scan error")
		}
	})

	t.Run("upsertRow SeenInFavorite branch", func(t *testing.T) {
		rawDB, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := NewDBUserArchiveService(wrapMySQLDB(rawDB))

		mock.ExpectQuery(`SELECT seen_in_history, seen_in_favorite FROM chat_user_archive WHERE owner_user_id = \? AND target_user_id = \? LIMIT 1`).
			WithArgs("me", "u9").
			WillReturnRows(sqlmock.NewRows([]string{"seen_in_history", "seen_in_favorite"}).AddRow(1, 0))
		mock.ExpectExec(`UPDATE chat_user_archive`).
			WithArgs(nil, nil, nil, 1, 1, sqlmock.AnyArg(), sqlmock.AnyArg(), "me", "u9").
			WillReturnResult(sqlmock.NewResult(0, 1))
		if err := svc.upsertRow(context.Background(), archiveUpsertInput{OwnerUserID: "me", TargetUserID: "u9", SeenInFavorite: 1}); err != nil {
			t.Fatalf("upsertRow err=%v", err)
		}
	})
}
