package app

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNormalizeStringList(t *testing.T) {
	in := []string{" a ", "", "b", "a", "  ", "b", "c"}
	got := normalizeStringList(in)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got=%v, want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%v, want=%v", got, want)
		}
	}
}

func TestNormalizeInt64List(t *testing.T) {
	in := []int64{0, 1, 2, 1, -3, 2, 9}
	got := normalizeInt64List(in)
	want := []int64{1, 2, 9}
	if len(got) != len(want) {
		t.Fatalf("got=%v, want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%v, want=%v", got, want)
		}
	}
}

func TestDouyinFavoriteService_ApplyUserTags_Set_DefaultAndDedup(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map WHERE sec_user_id = \?`).
		WithArgs("a").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
		WithArgs("a", int64(1), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
		WithArgs("a", int64(2), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map WHERE sec_user_id = \?`).
		WithArgs("b").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
		WithArgs("b", int64(1), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
		WithArgs("b", int64(2), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	if err := s.ApplyUserTags(context.Background(), []string{" a ", "a", "b"}, []int64{0, 1, 1, 2}, ""); err != nil {
		t.Fatalf("ApplyUserTags: %v", err)
	}
}

func TestDouyinFavoriteService_ApplyUserTags_InvalidMode(t *testing.T) {
	db := mustNewSQLMockDB(t)
	defer db.Close()

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	if err := s.ApplyUserTags(context.Background(), []string{"a"}, []int64{1}, "bad"); !errors.Is(err, ErrDouyinFavoriteTagInvalidMode) {
		t.Fatalf("err=%v, want %v", err, ErrDouyinFavoriteTagInvalidMode)
	}
}

func TestDouyinFavoriteService_ApplyUserTags_AddRemove_EmptyTagIDs(t *testing.T) {
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectCommit()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyUserTags(context.Background(), []string{"a"}, []int64{}, "add"); err != nil {
			t.Fatalf("add empty tagIDs: %v", err)
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectCommit()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyUserTags(context.Background(), []string{"a"}, []int64{}, "remove"); err != nil {
			t.Fatalf("remove empty tagIDs: %v", err)
		}
	}
}

func TestDouyinFavoriteService_ApplyUserTags_DeleteError_RollsBack(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map WHERE sec_user_id = \?`).
		WithArgs("a").
		WillReturnError(errors.New("delete failed"))
	mock.ExpectRollback()

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	if err := s.ApplyUserTags(context.Background(), []string{"a"}, []int64{1}, "set"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDouyinFavoriteService_ApplyAwemeTags_AddRemove(t *testing.T) {
	// add
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_aweme_tag_map`).
			WithArgs("aweme1", int64(7), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyAwemeTags(context.Background(), []string{"aweme1"}, []int64{7}, "add"); err != nil {
			t.Fatalf("ApplyAwemeTags add: %v", err)
		}
	}

	// remove
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map`).
			WithArgs("aweme1", int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyAwemeTags(context.Background(), []string{"aweme1"}, []int64{7}, "remove"); err != nil {
			t.Fatalf("ApplyAwemeTags remove: %v", err)
		}
	}
}

func TestDouyinFavoriteService_ApplyUserTags_AddAndRemoveBranches(t *testing.T) {
	// empty targetIDs => no tx
	{
		db := mustNewSQLMockDB(t)
		defer db.Close()
		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyUserTags(context.Background(), []string{" ", ""}, []int64{1}, "set"); err != nil {
			t.Fatalf("empty targets: %v", err)
		}
	}

	// add with non-empty tagIDs
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
			WithArgs("u1", int64(7), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT (IGNORE )?INTO douyin_favorite_user_tag_map`).
			WithArgs("u1", int64(8), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyUserTags(context.Background(), []string{"u1"}, []int64{7, 8}, "add"); err != nil {
			t.Fatalf("add: %v", err)
		}
	}

	// remove error -> rollback
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map`).
			WithArgs("u1", int64(7)).
			WillReturnError(errors.New("delete failed"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyUserTags(context.Background(), []string{"u1"}, []int64{7}, "remove"); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestDouyinFavoriteService_ApplyAwemeTags_EmptyTargetsAndEmptyTagIDs(t *testing.T) {
	// empty targetIDs => no tx
	{
		db := mustNewSQLMockDB(t)
		defer db.Close()
		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyAwemeTags(context.Background(), []string{" ", ""}, []int64{1}, "set"); err != nil {
			t.Fatalf("empty targets: %v", err)
		}
	}

	// add empty tagIDs => commit only
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectCommit()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyAwemeTags(context.Background(), []string{"a1"}, []int64{}, "add"); err != nil {
			t.Fatalf("add empty: %v", err)
		}
	}

	// remove empty tagIDs => commit only
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectCommit()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyAwemeTags(context.Background(), []string{"a1"}, []int64{}, "remove"); err != nil {
			t.Fatalf("remove empty: %v", err)
		}
	}

	// set delete error => rollback
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map WHERE aweme_id = \?`).
			WithArgs("a1").
			WillReturnError(errors.New("delete failed"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ApplyAwemeTags(context.Background(), []string{"a1"}, []int64{1}, "set"); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestDouyinFavoriteService_ListAllUserTagIDsAndFill(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_user_tag_map`).
		WillReturnRows(sqlmock.NewRows([]string{"sec_user_id", "tag_id"}).
			AddRow("u1", int64(1)).
			AddRow("", int64(2)).     // skip
			AddRow("u1", int64(0)).   // skip
			AddRow(" u2 ", int64(3)). // trim
			AddRow("u1", int64(9)))   // keep

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	m, err := s.listAllUserTagIDs(context.Background())
	if err != nil {
		t.Fatalf("listAllUserTagIDs: %v", err)
	}
	if got := len(m["u1"]); got != 2 {
		t.Fatalf("u1 tagIDs=%v, want 2 items", m["u1"])
	}
	if got := len(m["u2"]); got != 1 || m["u2"][0] != 3 {
		t.Fatalf("u2 tagIDs=%v, want [3]", m["u2"])
	}

	items := []DouyinFavoriteUser{
		{SecUserID: "u1"},
		{SecUserID: "u2"},
		{SecUserID: "u3"},
	}

	// fillUserTagIDs will query again.
	mock.ExpectQuery(`FROM douyin_favorite_user_tag_map`).
		WillReturnRows(sqlmock.NewRows([]string{"sec_user_id", "tag_id"}).
			AddRow("u1", int64(1)))

	if err := s.fillUserTagIDs(context.Background(), items); err != nil {
		t.Fatalf("fillUserTagIDs: %v", err)
	}
	if len(items[0].TagIDs) != 1 || items[0].TagIDs[0] != 1 {
		t.Fatalf("items[0].TagIDs=%v, want [1]", items[0].TagIDs)
	}
	if items[2].TagIDs == nil || len(items[2].TagIDs) != 0 {
		t.Fatalf("items[2].TagIDs=%v, want []", items[2].TagIDs)
	}
}

func TestDouyinFavoriteService_ListAllAwemeTagIDsAndFill(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_aweme_tag_map`).
		WillReturnRows(sqlmock.NewRows([]string{"aweme_id", "tag_id"}).
			AddRow("a1", int64(2)).
			AddRow(" ", int64(3)).   // skip
			AddRow("a1", int64(-1)). // skip
			AddRow("a2", int64(4)))

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	m, err := s.listAllAwemeTagIDs(context.Background())
	if err != nil {
		t.Fatalf("listAllAwemeTagIDs: %v", err)
	}
	if got := len(m["a1"]); got != 1 || m["a1"][0] != 2 {
		t.Fatalf("a1 tagIDs=%v, want [2]", m["a1"])
	}
	if got := len(m["a2"]); got != 1 || m["a2"][0] != 4 {
		t.Fatalf("a2 tagIDs=%v, want [4]", m["a2"])
	}

	items := []DouyinFavoriteAweme{
		{AwemeID: "a1"},
		{AwemeID: "a2"},
		{AwemeID: "a3"},
	}

	mock.ExpectQuery(`FROM douyin_favorite_aweme_tag_map`).
		WillReturnRows(sqlmock.NewRows([]string{"aweme_id", "tag_id"}).
			AddRow("a2", int64(8)))

	if err := s.fillAwemeTagIDs(context.Background(), items); err != nil {
		t.Fatalf("fillAwemeTagIDs: %v", err)
	}
	if len(items[1].TagIDs) != 1 || items[1].TagIDs[0] != 8 {
		t.Fatalf("items[1].TagIDs=%v, want [8]", items[1].TagIDs)
	}
	if items[2].TagIDs == nil || len(items[2].TagIDs) != 0 {
		t.Fatalf("items[2].TagIDs=%v, want []", items[2].TagIDs)
	}
}

func TestDouyinFavoriteService_ListUserTags_EmptyResultIsNotNilSlice(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_user_tag`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"sort_order",
			"cnt",
			"created_at",
			"updated_at",
		}))

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	out, err := s.ListUserTags(context.Background())
	if err != nil {
		t.Fatalf("ListUserTags: %v", err)
	}
	if out == nil || len(out) != 0 {
		t.Fatalf("out=%v, want empty slice", out)
	}
}

func TestDouyinFavoriteService_ListAwemeTags_EmptyResultIsNotNilSlice(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_aweme_tag`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"sort_order",
			"cnt",
			"created_at",
			"updated_at",
		}))

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	out, err := s.ListAwemeTags(context.Background())
	if err != nil {
		t.Fatalf("ListAwemeTags: %v", err)
	}
	if out == nil || len(out) != 0 {
		t.Fatalf("out=%v, want empty slice", out)
	}
}

func TestDouyinFavoriteService_RemoveUserTag_RollbackBranches(t *testing.T) {
	// fail at deleting map
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map WHERE tag_id = \?`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("delete map failed"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.RemoveUserTag(context.Background(), 1); err == nil {
			t.Fatalf("expected error")
		}
	}

	// fail at deleting tag row
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag_map WHERE tag_id = \?`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM douyin_favorite_user_tag WHERE id = \?`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("delete tag failed"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.RemoveUserTag(context.Background(), 1); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestDouyinFavoriteService_RemoveAwemeTag_RollbackBranches(t *testing.T) {
	// fail at deleting map
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map WHERE tag_id = \?`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("delete map failed"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.RemoveAwemeTag(context.Background(), 1); err == nil {
			t.Fatalf("expected error")
		}
	}

	// fail at deleting tag row
	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map WHERE tag_id = \?`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag WHERE id = \?`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("delete tag failed"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.RemoveAwemeTag(context.Background(), 1); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestDouyinFavoriteService_UpdateAwemeTag_Duplicate(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE douyin_favorite_aweme_tag`).
		WithArgs("教程", sqlmock.AnyArg(), int64(1)).
		WillReturnError(duplicateKeyErr())

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	_, err := s.UpdateAwemeTag(context.Background(), 1, "教程")
	if !errors.Is(err, ErrDouyinFavoriteTagAlreadyExists) {
		t.Fatalf("err=%v, want %v", err, ErrDouyinFavoriteTagAlreadyExists)
	}
}

func TestDouyinFavoriteService_FindAwemeTagByID_NoRows(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM douyin_favorite_aweme_tag t`).
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "sort_order", "cnt", "created_at", "updated_at",
		}))

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	out, err := s.findAwemeTagByID(context.Background(), 123)
	if err != nil {
		t.Fatalf("findAwemeTagByID: %v", err)
	}
	if out != nil {
		t.Fatalf("out=%v, want nil", out)
	}
}

func TestDouyinFavoriteService_ReorderAwemeTags_UpdateErrorRollback(t *testing.T) {
	{
		db := mustNewSQLMockDB(t)
		defer db.Close()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ReorderAwemeTags(context.Background(), []int64{}); err != nil {
			t.Fatalf("empty reorder: %v", err)
		}
	}

	{
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE douyin_favorite_aweme_tag SET sort_order = \? WHERE id = \?`).
			WithArgs(int64(1), int64(2)).
			WillReturnError(errors.New("update failed"))
		mock.ExpectRollback()

		s := NewDouyinFavoriteService(wrapMySQLDB(db))
		if err := s.ReorderAwemeTags(context.Background(), []int64{2}); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestDouyinFavoriteService_ApplyAwemeTags_Set_EmptyTagIDs(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM douyin_favorite_aweme_tag_map WHERE aweme_id = \?`).
		WithArgs("a1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	s := NewDouyinFavoriteService(wrapMySQLDB(db))
	if err := s.ApplyAwemeTags(context.Background(), []string{"a1"}, []int64{}, "set"); err != nil {
		t.Fatalf("ApplyAwemeTags set empty: %v", err)
	}
}
