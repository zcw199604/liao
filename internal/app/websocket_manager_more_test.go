package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestToInt(t *testing.T) {
	if got := toInt(1); got != 1 {
		t.Fatalf("got=%d", got)
	}
	if got := toInt(int64(2)); got != 2 {
		t.Fatalf("got=%d", got)
	}
	if got := toInt(float64(3)); got != 3 {
		t.Fatalf("got=%d", got)
	}
	if got := toInt(json.Number("4")); got != 4 {
		t.Fatalf("got=%d", got)
	}
	if got := toInt(json.Number("x")); got != 0 {
		t.Fatalf("got=%d", got)
	}
	if got := toInt(nil); got != 0 {
		t.Fatalf("got=%d", got)
	}
}

func TestToBool(t *testing.T) {
	if got := toBool(true); got != true {
		t.Fatalf("got=%v", got)
	}
	if got := toBool(" TRUE "); got != true {
		t.Fatalf("got=%v", got)
	}
	if got := toBool("1"); got != true {
		t.Fatalf("got=%v", got)
	}
	if got := toBool("0"); got != false {
		t.Fatalf("got=%v", got)
	}
	if got := toBool(float64(2)); got != true {
		t.Fatalf("got=%v", got)
	}
	if got := toBool(float64(0)); got != false {
		t.Fatalf("got=%v", got)
	}
	if got := toBool(nil); got != false {
		t.Fatalf("got=%v", got)
	}
}

func TestMapGetMap(t *testing.T) {
	if got := mapGetMap(nil, "x"); len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
	if got := mapGetMap(map[string]any{}, "x"); len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
	if got := mapGetMap(map[string]any{"x": nil}, "x"); len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
	if got := mapGetMap(map[string]any{"x": "y"}, "x"); len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
	mm := map[string]any{"k": "v"}
	if got := mapGetMap(map[string]any{"x": mm}, "x"); got["k"] != "v" {
		t.Fatalf("got=%v", got)
	}
}

func TestUpstreamWebSocketManager_getUpstreamWebSocketURL(t *testing.T) {
	t.Run("nil httpClient returns fallback", func(t *testing.T) {
		m := NewUpstreamWebSocketManager(nil, "ws://fallback", nil, nil, nil)
		if got := m.getUpstreamWebSocketURL(t.Context()); got != "ws://fallback" {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("do error returns fallback", func(t *testing.T) {
		c := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("do fail")
		})}
		m := NewUpstreamWebSocketManager(c, "ws://fallback", nil, nil, nil)
		if got := m.getUpstreamWebSocketURL(t.Context()); got != "ws://fallback" {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("new request error returns fallback", func(t *testing.T) {
		oldBase := wsWebServiceRandServerBase
		wsWebServiceRandServerBase = "http://[::1"
		t.Cleanup(func() { wsWebServiceRandServerBase = oldBase })

		c := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			t.Fatalf("should not reach RoundTrip when request build fails")
			return nil, nil
		})}
		m := NewUpstreamWebSocketManager(c, "ws://fallback", nil, nil, nil)
		if got := m.getUpstreamWebSocketURL(t.Context()); got != "ws://fallback" {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("read error returns fallback", func(t *testing.T) {
		c := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       errReadCloser{},
				Header:     make(http.Header),
			}, nil
		})}
		m := NewUpstreamWebSocketManager(c, "ws://fallback", nil, nil, nil)
		if got := m.getUpstreamWebSocketURL(t.Context()); got != "ws://fallback" {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("json unmarshal error returns fallback", func(t *testing.T) {
		c := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString("not-json")),
				Header:     make(http.Header),
			}, nil
		})}
		m := NewUpstreamWebSocketManager(c, "ws://fallback", nil, nil, nil)
		if got := m.getUpstreamWebSocketURL(t.Context()); got != "ws://fallback" {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("state not OK returns fallback", func(t *testing.T) {
		c := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(`{"state":"NO","msg":{"server":"ws://x"}}`)),
				Header:     make(http.Header),
			}, nil
		})}
		m := NewUpstreamWebSocketManager(c, "ws://fallback", nil, nil, nil)
		if got := m.getUpstreamWebSocketURL(t.Context()); got != "ws://fallback" {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("empty server returns fallback", func(t *testing.T) {
		c := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(`{"state":"OK","msg":{"server":"  "}}`)),
				Header:     make(http.Header),
			}, nil
		})}
		m := NewUpstreamWebSocketManager(c, "ws://fallback", nil, nil, nil)
		if got := m.getUpstreamWebSocketURL(t.Context()); got != "ws://fallback" {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("success returns server", func(t *testing.T) {
		c := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(bytes.NewBufferString(`{"state":"OK","msg":{"server":"ws://good"}}`)),
				Header:     make(http.Header),
			}, nil
		})}
		m := NewUpstreamWebSocketManager(c, "ws://fallback", nil, nil, nil)
		if got := m.getUpstreamWebSocketURL(t.Context()); got != "ws://good" {
			t.Fatalf("got=%q", got)
		}
	})
}
