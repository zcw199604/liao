package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMax64(t *testing.T) {
	if max64(2, 1) != 2 {
		t.Fatalf("expected 2")
	}
	if max64(1, 2) != 2 {
		t.Fatalf("expected 2")
	}
	if max64(1, 1) != 1 {
		t.Fatalf("expected 1")
	}
}

func TestMediaUploadService_RepairMediaHistory_InvalidLimits(t *testing.T) {
	svc := &MediaUploadService{}
	if _, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{LimitMissingMD5: -1}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_RepairMediaHistory_DryRunRunsOnceAndWarns(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	oldCalc := calculateMD5FromLocalPathFn
	calculateMD5FromLocalPathFn = func(_ *FileStorageService, localPath string) (string, error) {
		switch localPath {
		case "err":
			return "", fmt.Errorf("md5 err")
		default:
			return "m1", nil
		}
	}
	t.Cleanup(func() { calculateMD5FromLocalPathFn = oldCalc })

	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}).
			AddRow(int64(1), "u1", "ok"))

	mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}).
			AddRow("m1", int64(2)))

	mock.ExpectQuery(`(?s)SELECT id, user_id.*FROM media_upload_history.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs("m1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
			AddRow(int64(10), "keep"))

	svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{}}
	res, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{
		Commit:             false,
		UserID:             " u1 ",
		FixMissingMD5:      false,
		DeduplicateByMD5:   false,
		SampleLimit:        0,
		LimitMissingMD5:    0,
		MaxDuplicateGroups: 0,
	})
	if err != nil {
		t.Fatalf("RepairMediaHistory error: %v", err)
	}
	if !res.FixMissingMD5 || !res.DeduplicateByMD5 {
		t.Fatalf("expected FixMissingMD5/DeduplicateByMD5 forced true, got %+v", res)
	}
	if res.MissingMD5.Scanned != 1 || res.MissingMD5.NeedUpdate != 1 {
		t.Fatalf("missingMD5=%+v", res.MissingMD5)
	}
	if len(res.Warnings) < 3 {
		t.Fatalf("warnings=%v", res.Warnings)
	}
}

func TestMediaUploadService_RepairMediaHistory_CommitLoopsAndNoProgressBreak(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	oldCalc := calculateMD5FromLocalPathFn
	calculateMD5FromLocalPathFn = func(_ *FileStorageService, localPath string) (string, error) {
		return "m1", nil
	}
	t.Cleanup(func() { calculateMD5FromLocalPathFn = oldCalc })

	// missing md5: first batch has 1 row, second batch empty.
	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}).
			AddRow(int64(1), "u1", "ok"))
	mock.ExpectExec(`(?s)UPDATE media_upload_history.*SET file_md5 = [?].*WHERE id = [?].*`).
		WithArgs("m1", int64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(1), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}))

	// md5 dedup: one group but delete affects 0 rows => no progress => break.
	mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}).
			AddRow("m1", int64(2)))
	mock.ExpectQuery(`(?s)SELECT id, user_id.*FROM media_upload_history.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs("m1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
			AddRow(int64(10), "keep"))
	mock.ExpectExec(`(?s)DELETE FROM media_upload_history.*WHERE file_md5 = [?] AND id <> [?]`).
		WithArgs("m1", int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{}}
	res, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{Commit: true})
	if err != nil {
		t.Fatalf("RepairMediaHistory error: %v", err)
	}
	if res.MissingMD5.Updated != 1 {
		t.Fatalf("missingMD5=%+v", res.MissingMD5)
	}
	if len(res.Warnings) == 0 {
		t.Fatalf("expected warnings")
	}
}

func TestMediaUploadService_RepairMediaHistory_CommitDedupProgressThenFinish(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// missing md5 batch: no rows => break quickly.
	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}))

	// md5 dedup: 1st iteration deletes 1 row; 2nd iteration has no groups.
	mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}).
			AddRow("m1", int64(2)))
	mock.ExpectQuery(`(?s)SELECT id, user_id.*FROM media_upload_history.*WHERE file_md5 = [?].*LIMIT 1`).
		WithArgs("m1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
			AddRow(int64(10), "keep"))
	mock.ExpectExec(`(?s)DELETE FROM media_upload_history.*WHERE file_md5 = [?] AND id <> [?]`).
		WithArgs("m1", int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}))

	svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{}}
	res, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{Commit: true})
	if err != nil {
		t.Fatalf("RepairMediaHistory error: %v", err)
	}
	if res.DuplicatesByMD5.Deleted != 1 {
		t.Fatalf("dedup=%+v", res.DuplicatesByMD5)
	}
}

func TestMediaUploadService_RepairMediaHistory_SampleLimitClampedTo200(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// missing md5 batch: no rows.
	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}))

	groups := sqlmock.NewRows([]string{"file_md5", "cnt"})
	for i := 0; i < 205; i++ {
		groups.AddRow(fmt.Sprintf("m%d", i), int64(2))
	}
	mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(groups)

	for i := 0; i < 205; i++ {
		md5v := fmt.Sprintf("m%d", i)
		mock.ExpectQuery(`(?s)SELECT id, user_id.*FROM media_upload_history.*WHERE file_md5 = [?].*LIMIT 1`).
			WithArgs(md5v).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
				AddRow(int64(10+i), fmt.Sprintf("u%d", i)))
		// commit=false => no delete exec.
	}

	svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{}}
	res, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{SampleLimit: 999})
	if err != nil {
		t.Fatalf("RepairMediaHistory error: %v", err)
	}
	if len(res.Samples) != 200 {
		t.Fatalf("samples=%d, want 200 (clamped)", len(res.Samples))
	}
}

func TestMediaUploadService_repairMissingMD5Batch_SkipsOnCalculateErrorAndEmptyMD5(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	oldCalc := calculateMD5FromLocalPathFn
	calculateMD5FromLocalPathFn = func(_ *FileStorageService, localPath string) (string, error) {
		switch localPath {
		case "err":
			return "", fmt.Errorf("boom")
		case "empty":
			return "", nil
		default:
			return "m", nil
		}
	}
	t.Cleanup(func() { calculateMD5FromLocalPathFn = oldCalc })

	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}).
			AddRow(int64(1), "u1", "err").
			AddRow(int64(2), "u2", "empty").
			AddRow(int64(3), "u3", "ok"))

	svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{}}
	req := &RepairMediaHistoryRequest{Commit: false, LimitMissingMD5: 10}
	res := &RepairMediaHistoryResult{}
	scanned, nextID, err := svc.repairMissingMD5Batch(context.Background(), 0, req, res)
	if err != nil {
		t.Fatalf("repairMissingMD5Batch error: %v", err)
	}
	if scanned != 3 || nextID != 3 {
		t.Fatalf("scanned/next=%d/%d, want 3/3", scanned, nextID)
	}
	if res.MissingMD5.Skipped != 2 || res.MissingMD5.NeedUpdate != 1 || res.MissingMD5.Updated != 0 {
		t.Fatalf("result=%+v", res.MissingMD5)
	}
}

func TestMediaUploadService_repairMissingMD5Batch_CommitUpdateSuccessAndWarnOnUpdateError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	oldCalc := calculateMD5FromLocalPathFn
	calculateMD5FromLocalPathFn = func(_ *FileStorageService, localPath string) (string, error) {
		if localPath == "ok1" {
			return "m1", nil
		}
		return "m2", nil
	}
	t.Cleanup(func() { calculateMD5FromLocalPathFn = oldCalc })

	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}).
			AddRow(int64(1), "u1", "ok1").
			AddRow(int64(2), "u2", "ok2"))

	mock.ExpectExec(`(?s)UPDATE media_upload_history.*SET file_md5 = [?].*WHERE id = [?].*`).
		WithArgs("m1", int64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)UPDATE media_upload_history.*SET file_md5 = [?].*WHERE id = [?].*`).
		WithArgs("m2", int64(2)).
		WillReturnError(fmt.Errorf("update failed"))

	svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{}}
	req := &RepairMediaHistoryRequest{Commit: true, LimitMissingMD5: 10}
	res := &RepairMediaHistoryResult{}
	scanned, nextID, err := svc.repairMissingMD5Batch(context.Background(), 0, req, res)
	if err != nil {
		t.Fatalf("repairMissingMD5Batch error: %v", err)
	}
	if scanned != 2 || nextID != 2 {
		t.Fatalf("scanned/next=%d/%d, want 2/2", scanned, nextID)
	}
	if res.MissingMD5.Updated != 1 {
		t.Fatalf("updated=%d, want 1", res.MissingMD5.Updated)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "update md5 failed") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected update warning, got %v", res.Warnings)
	}
}

func TestMediaUploadService_repairMissingMD5Batch_QueryError_ScanError_RowsErr(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
			WithArgs(int64(0), 10).
			WillReturnError(fmt.Errorf("qerr"))

		svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{}}
		_, _, err := svc.repairMissingMD5Batch(context.Background(), 0, &RepairMediaHistoryRequest{LimitMissingMD5: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "query missing md5 rows") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
			WithArgs(int64(0), 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}).
				AddRow("bad", "u1", "ok"))

		svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{}}
		_, _, err := svc.repairMissingMD5Batch(context.Background(), 0, &RepairMediaHistoryRequest{LimitMissingMD5: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "scan missing md5 row") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("rows err", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		oldCalc := calculateMD5FromLocalPathFn
		calculateMD5FromLocalPathFn = func(_ *FileStorageService, localPath string) (string, error) {
			return "", fmt.Errorf("skip")
		}
		t.Cleanup(func() { calculateMD5FromLocalPathFn = oldCalc })

		rows := sqlmock.NewRows([]string{"id", "user_id", "local_path"}).
			AddRow(int64(1), "u1", "ok").
			AddRow(int64(2), "u2", "ok2").
			RowError(1, fmt.Errorf("rowerr"))
		mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
			WithArgs(int64(0), 10).
			WillReturnRows(rows)

		svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{}}
		_, _, err := svc.repairMissingMD5Batch(context.Background(), 0, &RepairMediaHistoryRequest{LimitMissingMD5: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "iterate missing md5 rows") {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestMediaUploadService_dedupByMD5Batch_ErrorsAndWarnings(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnError(fmt.Errorf("qerr"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		_, err := svc.dedupByMD5Batch(context.Background(), &RepairMediaHistoryRequest{MaxDuplicateGroups: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "query duplicate md5 groups") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}).
				AddRow("m1", "bad"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		_, err := svc.dedupByMD5Batch(context.Background(), &RepairMediaHistoryRequest{MaxDuplicateGroups: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "scan duplicate md5 group") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("select keep no rows and rows err", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{"file_md5", "cnt"}).
			AddRow("m1", int64(2)).
			AddRow("m2", int64(2)).
			RowError(1, fmt.Errorf("rowerr"))
		mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnRows(rows)

		mock.ExpectQuery(`(?s)SELECT id, user_id.*WHERE file_md5 = [?].*LIMIT 1`).
			WithArgs("m1").
			WillReturnError(sql.ErrNoRows)

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		_, err := svc.dedupByMD5Batch(context.Background(), &RepairMediaHistoryRequest{MaxDuplicateGroups: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "iterate duplicate md5 groups") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("select keep error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}).
				AddRow("m1", int64(2)))

		mock.ExpectQuery(`(?s)SELECT id, user_id.*WHERE file_md5 = [?].*LIMIT 1`).
			WithArgs("m1").
			WillReturnError(errors.New("boom"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		_, err := svc.dedupByMD5Batch(context.Background(), &RepairMediaHistoryRequest{MaxDuplicateGroups: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "select keep row for md5 dedup") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("commit delete error adds warning", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}).
				AddRow("m1", int64(3)))
		mock.ExpectQuery(`(?s)SELECT id, user_id.*WHERE file_md5 = [?].*LIMIT 1`).
			WithArgs("m1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
				AddRow(int64(10), "u1"))
		mock.ExpectExec(`(?s)DELETE FROM media_upload_history.*WHERE file_md5 = [?] AND id <> [?]`).
			WithArgs("m1", int64(10)).
			WillReturnError(errors.New("del"))

		res := &RepairMediaHistoryResult{}
		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		groups, err := svc.dedupByMD5Batch(context.Background(), &RepairMediaHistoryRequest{Commit: true, MaxDuplicateGroups: 10, SampleLimit: 1}, res)
		if err != nil {
			t.Fatalf("dedup error: %v", err)
		}
		if groups != 1 || res.DuplicatesByMD5.ToDelete != 2 {
			t.Fatalf("res=%+v", res.DuplicatesByMD5)
		}
		found := false
		for _, w := range res.Warnings {
			if strings.Contains(w, "md5 dedup delete failed") {
				found = true
			}
		}
		if !found {
			t.Fatalf("warnings=%v", res.Warnings)
		}
	})
}

func TestMediaUploadService_dedupByLocalPathBatch_ErrorsAndWarnings(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnError(fmt.Errorf("qerr"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		_, err := svc.dedupByLocalPathBatch(context.Background(), &RepairMediaHistoryRequest{MaxDuplicateGroups: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "query duplicate local_path groups") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"local_path", "cnt"}).
				AddRow("p1", "bad"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		_, err := svc.dedupByLocalPathBatch(context.Background(), &RepairMediaHistoryRequest{MaxDuplicateGroups: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "scan duplicate local_path group") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("select keep no rows and rows err", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{"local_path", "cnt"}).
			AddRow("p1", int64(2)).
			AddRow("p2", int64(2)).
			RowError(1, fmt.Errorf("rowerr"))
		mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnRows(rows)

		mock.ExpectQuery(`(?s)SELECT id, user_id.*WHERE local_path = [?].*LIMIT 1`).
			WithArgs("p1").
			WillReturnError(sql.ErrNoRows)

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		_, err := svc.dedupByLocalPathBatch(context.Background(), &RepairMediaHistoryRequest{MaxDuplicateGroups: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "iterate duplicate local_path groups") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("select keep error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"local_path", "cnt"}).
				AddRow("p1", int64(2)))

		mock.ExpectQuery(`(?s)SELECT id, user_id.*WHERE local_path = [?].*LIMIT 1`).
			WithArgs("p1").
			WillReturnError(errors.New("boom"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		_, err := svc.dedupByLocalPathBatch(context.Background(), &RepairMediaHistoryRequest{MaxDuplicateGroups: 10}, &RepairMediaHistoryResult{})
		if err == nil || !strings.Contains(err.Error(), "select keep row for local_path dedup") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("commit delete error adds warning", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"local_path", "cnt"}).
				AddRow("p1", int64(3)))
		mock.ExpectQuery(`(?s)SELECT id, user_id.*WHERE local_path = [?].*LIMIT 1`).
			WithArgs("p1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
				AddRow(int64(10), "u1"))
		mock.ExpectExec(`(?s)DELETE FROM media_upload_history.*WHERE local_path = [?].*id <> [?]`).
			WithArgs("p1", int64(10)).
			WillReturnError(errors.New("del"))

		res := &RepairMediaHistoryResult{}
		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		groups, err := svc.dedupByLocalPathBatch(context.Background(), &RepairMediaHistoryRequest{Commit: true, MaxDuplicateGroups: 10, SampleLimit: 1}, res)
		if err != nil {
			t.Fatalf("dedup error: %v", err)
		}
		if groups != 1 || res.DuplicatesByLocalPath.ToDelete != 2 {
			t.Fatalf("res=%+v", res.DuplicatesByLocalPath)
		}
		found := false
		for _, w := range res.Warnings {
			if strings.Contains(w, "local_path dedup delete failed") {
				found = true
			}
		}
		if !found {
			t.Fatalf("warnings=%v", res.Warnings)
		}
	})
}

func TestMediaUploadService_RepairMediaHistory_PropagatesBatchErrors(t *testing.T) {
	t.Run("repairMissingMD5Batch error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
			WithArgs(int64(0), 500).
			WillReturnError(errors.New("qerr"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		if _, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{Commit: true}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("dedupByMD5Batch error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
			WithArgs(int64(0), 500).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}))

		mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
			WithArgs(500).
			WillReturnError(errors.New("qerr"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		if _, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{Commit: true}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("dedupByLocalPathBatch error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
			WithArgs(int64(0), 500).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}))

		mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
			WithArgs(500).
			WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}))

		mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
			WithArgs(500).
			WillReturnError(errors.New("qerr"))

		svc := &MediaUploadService{db: wrapMySQLDB(db)}
		if _, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{Commit: true, DeduplicateByLocalPath: true}); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestMediaUploadService_RepairMediaHistory_DedupByLocalPath_NoGroups_Breaks(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}))

	mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}))

	mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"local_path", "cnt"}))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	if _, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{Commit: true, DeduplicateByLocalPath: true}); err != nil {
		t.Fatalf("RepairMediaHistory error: %v", err)
	}
}

func TestCalculateMD5FromLocalPathFn_Default(t *testing.T) {
	root := t.TempDir()
	fileStore := &FileStorageService{baseUploadAbs: root}

	path := "/images/a.txt"
	full := filepath.Join(root, "images", "a.txt")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("abc"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	md5Value, err := calculateMD5FromLocalPathFn(fileStore, path)
	if err != nil {
		t.Fatalf("calc: %v", err)
	}
	if strings.TrimSpace(md5Value) == "" {
		t.Fatalf("expected md5")
	}
}

func TestMediaUploadService_RepairMediaHistory_DedupByLocalPath_DryRunWarns(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// missing md5 batch: no rows.
	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}))

	// md5 dedup: no groups.
	mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}))

	// local_path dedup: 1 group.
	mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"local_path", "cnt"}).
			AddRow("p1", int64(2)))

	mock.ExpectQuery(`(?s)SELECT id, user_id.*WHERE local_path = [?].*LIMIT 1`).
		WithArgs("p1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
			AddRow(int64(10), "u1"))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	res, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{
		Commit:                 false,
		FixMissingMD5:          true,
		DeduplicateByMD5:       true,
		DeduplicateByLocalPath: true,
	})
	if err != nil {
		t.Fatalf("RepairMediaHistory error: %v", err)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "local_path 分组") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected local_path dry-run warning, warnings=%v", res.Warnings)
	}
}

func TestMediaUploadService_RepairMediaHistory_DedupByLocalPath_NoProgressBreakWarns(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// missing md5 batch: no rows.
	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}))

	// md5 dedup: no groups.
	mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}))

	// local_path dedup: 1 group, but delete makes no progress.
	mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"local_path", "cnt"}).
			AddRow("p1", int64(2)))

	mock.ExpectQuery(`(?s)SELECT id, user_id.*WHERE local_path = [?].*LIMIT 1`).
		WithArgs("p1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
			AddRow(int64(10), "u1"))

	mock.ExpectExec(`(?s)DELETE FROM media_upload_history.*WHERE local_path = [?].*id <> [?]`).
		WithArgs("p1", int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	res, err := svc.RepairMediaHistory(context.Background(), RepairMediaHistoryRequest{
		Commit:                 true,
		FixMissingMD5:          true,
		DeduplicateByMD5:       true,
		DeduplicateByLocalPath: true,
	})
	if err != nil {
		t.Fatalf("RepairMediaHistory error: %v", err)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "local_path 去重未产生删除进展") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected local_path no-progress warning, warnings=%v", res.Warnings)
	}
}

func TestMediaUploadService_dedupByLocalPathBatch_CommitDeleteIncrementsDeleted(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT local_path, COUNT.*FROM media_upload_history.*LIMIT [?]`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"local_path", "cnt"}).
			AddRow("p1", int64(3)))

	mock.ExpectQuery(`(?s)SELECT id, user_id.*WHERE local_path = [?].*LIMIT 1`).
		WithArgs("p1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).
			AddRow(int64(10), "u1"))

	mock.ExpectExec(`(?s)DELETE FROM media_upload_history.*WHERE local_path = [?].*id <> [?]`).
		WithArgs("p1", int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 2))

	res := &RepairMediaHistoryResult{}
	svc := &MediaUploadService{db: wrapMySQLDB(db)}
	groups, err := svc.dedupByLocalPathBatch(context.Background(), &RepairMediaHistoryRequest{Commit: true, MaxDuplicateGroups: 10, SampleLimit: 1}, res)
	if err != nil {
		t.Fatalf("dedup error: %v", err)
	}
	if groups != 1 || res.DuplicatesByLocalPath.Deleted != 2 {
		t.Fatalf("res=%+v", res.DuplicatesByLocalPath)
	}
}
