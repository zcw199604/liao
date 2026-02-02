package app

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"

	"liao/internal/database"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))
}

func newMultipartRequest(t *testing.T, method, url, fieldName, filename, contentType string, content []byte, formValues map[string]string) (*http.Request, *multipart.FileHeader) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, v := range formValues {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("写入表单字段失败: %v", err)
		}
	}

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filename))
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("创建文件 part 失败: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("写入文件内容失败: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("关闭 multipart writer 失败: %v", err)
	}

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if err := req.ParseMultipartForm(32 << 20); err != nil {
		t.Fatalf("解析 multipart form 失败: %v", err)
	}

	files := req.MultipartForm.File[fieldName]
	if len(files) == 0 {
		t.Fatalf("未找到上传文件字段: %s", fieldName)
	}
	return req, files[0]
}

func newSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(placeholderNormalizingMatcher{}))
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}

	cleanup := func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sqlmock 期望未满足: %v", err)
		}
		_ = db.Close()
	}

	return db, mock, cleanup
}

func mustNewSQLMockDB(t *testing.T) *sql.DB {
	t.Helper()

	db, _, err := sqlmock.New(sqlmock.QueryMatcherOption(placeholderNormalizingMatcher{}))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	return db
}

var placeholderDollarRe = regexp.MustCompile(`\\$\\d+`)

// placeholderNormalizingMatcher makes sqlmock expectations reusable for both MySQL ('?')
// and Postgres ('$1..$n') placeholder styles by normalizing actual queries.
//
// This lets most tests keep using '?' patterns even when the DB wrapper rebinding is enabled.
type placeholderNormalizingMatcher struct{}

func (placeholderNormalizingMatcher) Match(expectedSQL, actualSQL string) error {
	normalized := placeholderDollarRe.ReplaceAllString(actualSQL, "?")
	return sqlmock.QueryMatcherRegexp.Match(expectedSQL, normalized)
}

// For sqlmock-based tests, keep placeholders as '?' even in "postgres" mode.
// The real runtime path rebinding to '$1..$n' is covered by internal/database tests.
type testPostgresDialect struct{ database.PostgresDialect }

func (testPostgresDialect) Rebind(query string) string { return query }

func testDialect() database.Dialect {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("TEST_DB_DIALECT")))
	switch v {
	case "postgres", "postgresql", "pg":
		return testPostgresDialect{}
	default:
		return database.MySQLDialect{}
	}
}

func wrapMySQLDB(db *sql.DB) *database.DB {
	return database.Wrap(db, testDialect())
}

func expectInsertReturningID(mock sqlmock.Sqlmock, query string, id int64, args ...driver.Value) {
	if testDialect().Name() == "postgres" {
		rows := sqlmock.NewRows([]string{"id"}).AddRow(id)
		mock.ExpectQuery(query).WithArgs(args...).WillReturnRows(rows)
		return
	}
	mock.ExpectExec(query).WithArgs(args...).WillReturnResult(sqlmock.NewResult(id, 1))
}

func expectInsertReturningIDError(mock sqlmock.Sqlmock, query string, err error, args ...driver.Value) {
	if testDialect().Name() == "postgres" {
		mock.ExpectQuery(query).WithArgs(args...).WillReturnError(err)
		return
	}
	mock.ExpectExec(query).WithArgs(args...).WillReturnError(err)
}

func duplicateKeyErr() error {
	if testDialect().Name() == "postgres" {
		return &pgconn.PgError{Code: "23505"}
	}
	return &mysql.MySQLError{Number: 1062, Message: "dup"}
}
