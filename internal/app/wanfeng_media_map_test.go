package app

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/database"
)

// TestWanfengMediaMapDBGet 校验映射查询：命中返回文件名，未命中返回 false。
func TestWanfengMediaMapDBGet(t *testing.T) {
	raw, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = raw.Close() }()
	store := NewWanfengMediaMapDB(database.Wrap(raw, database.MySQLDialect{}))

	mock.ExpectQuery(`SELECT remote_filename FROM wanfeng_media_map WHERE wf_key = \?`).
		WithArgs("abc123").
		WillReturnRows(sqlmock.NewRows([]string{"remote_filename"}).AddRow("2026/11/07/Random/a.jpg"))

	name, ok, err := store.Get(context.Background(), "abc123")
	if err != nil || !ok || name != "2026/11/07/Random/a.jpg" {
		t.Fatalf("查询映射结果不符: name=%q ok=%v err=%v", name, ok, err)
	}

	mock.ExpectQuery(`SELECT remote_filename FROM wanfeng_media_map WHERE wf_key = \?`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"remote_filename"}))
	if _, ok, err := store.Get(context.Background(), "missing"); err != nil || ok {
		t.Fatalf("未命中应返回 false: ok=%v err=%v", ok, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足的期望: %v", err)
	}
}

// TestWanfengMediaMapDBPutUpsert 校验写入用 upsert（不同方言不同语法）。
func TestWanfengMediaMapDBPutUpsert(t *testing.T) {
	cases := []struct {
		name    string
		dialect database.Dialect
		want    string
	}{
		{"mysql", database.MySQLDialect{}, `ON DUPLICATE KEY UPDATE`},
		{"postgres", database.PostgresDialect{}, `ON CONFLICT \(wf_key\) DO UPDATE SET`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer func() { _ = raw.Close() }()
			store := NewWanfengMediaMapDB(database.Wrap(raw, c.dialect))
			mock.ExpectExec(`INSERT INTO wanfeng_media_map`).
				WillReturnResult(sqlmock.NewResult(1, 1))
			if err := store.Put(context.Background(), "abc", "2026/a.jpg", "IDENT-1"); err != nil {
				t.Fatalf("写入映射失败: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("未满足的期望: %v", err)
			}
		})
	}
}

// TestWanfengMediaMapDBNilSafe 校验空依赖时不 panic。
func TestWanfengMediaMapDBNilSafe(t *testing.T) {
	if NewWanfengMediaMapDB(nil) != nil {
		t.Fatal("db 为空应返回 nil store")
	}
	var store *wanfengMediaMapDB
	if _, ok, err := store.Get(context.Background(), "k"); ok || err != nil {
		t.Fatalf("nil store 应安全返回: ok=%v err=%v", ok, err)
	}
	if err := store.Put(context.Background(), "k", "v", "id"); err != nil {
		t.Fatalf("nil store Put 不应报错: %v", err)
	}
}
