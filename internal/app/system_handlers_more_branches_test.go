package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestHandleBatchDeleteUpstreamUsers_MoreBranches(t *testing.T) {
	t.Run("decode error", func(t *testing.T) {
		a := &App{}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/batchDeleteUpstreamUsers", strings.NewReader("{"))
		rec := httptest.NewRecorder()
		a.handleBatchDeleteUpstreamUsers(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("dedup leaves empty list", func(t *testing.T) {
		a := &App{}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/batchDeleteUpstreamUsers", strings.NewReader(`{"myUserId":"1","userToIds":[" ","\t"]}`))
		rec := httptest.NewRecorder()
		a.handleBatchDeleteUpstreamUsers(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("too many ids", func(t *testing.T) {
		ids := make([]string, 0, 201)
		for i := 0; i < 201; i++ {
			ids = append(ids, `"u`+strconv.Itoa(i)+`"`)
		}
		payload := `{"myUserId":"1","userToIds":[` + strings.Join(ids, ",") + `]}`
		a := &App{}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/batchDeleteUpstreamUsers", strings.NewReader(payload))
		rec := httptest.NewRecorder()
		a.handleBatchDeleteUpstreamUsers(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("request build error in loop", func(t *testing.T) {
		old := newHTTPRequestWithContextFn
		t.Cleanup(func() { newHTTPRequestWithContextFn = old })
		newHTTPRequestWithContextFn = func(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
			return nil, io.ErrUnexpectedEOF
		}

		a := &App{}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/batchDeleteUpstreamUsers", bytes.NewBufferString(`{"myUserId":"1","userToIds":["u1"]}`))
		rec := httptest.NewRecorder()
		a.handleBatchDeleteUpstreamUsers(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
	})
}
