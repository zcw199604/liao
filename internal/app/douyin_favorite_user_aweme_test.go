package app

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDouyinFavoriteService_UpdateUserAwemeDownloadsCover(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE douyin_favorite_user_aweme`).
		WithArgs(sqlmock.AnyArg(), "https://example.com/c.jpg", "u1", "a1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := NewDouyinFavoriteService(wrapMySQLDB(db))
	if err := svc.UpdateUserAwemeDownloadsCover(context.Background(), " u1 ", " a1 ", []string{" https://example.com/v.mp4 "}, " https://example.com/c.jpg "); err != nil {
		t.Fatalf("err=%v", err)
	}
}
