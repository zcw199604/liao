package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"liao/internal/config"
)

func TestWorkerPreviewRequiresLiaoLogin(t *testing.T) {
	a := &App{jwt: NewJWTService("test-secret", 1), staticDir: t.TempDir()}
	for _, endpoint := range []struct{ method, path string }{
		{http.MethodPost, "/api/workerPreview/session"},
		{http.MethodGet, "/worker-preview/preview"},
		{http.MethodGet, "/worker-preview/api/media?url=https://pbs.twimg.com/image.jpg"},
	} {
		rr := httptest.NewRecorder()
		a.buildRouter().ServeHTTP(rr, httptest.NewRequest(endpoint.method, endpoint.path, nil))
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status=%d, want 401", endpoint.method, endpoint.path, rr.Code)
		}
	}
}

func newWorkerPreviewTestApp(t *testing.T, handler http.HandlerFunc) (*App, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cfg := config.Config{JWTSecret: "liao-test-secret", TokenExpireHours: 1, WorkerPreviewURL: server.URL, WorkerPreviewUsername: "preview-user", WorkerPreviewPassword: "preview-secret"}
	a := &App{cfg: cfg, jwt: NewJWTService(cfg.JWTSecret, 1), staticDir: t.TempDir()}
	a.workerPreview = newWorkerPreviewService(cfg, server.Client())
	return a, server
}

func createWorkerPreviewTestSession(t *testing.T, a *App) *http.Cookie {
	t.Helper()
	token, err := a.jwt.GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/workerPreview/session", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	a.buildRouter().ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("create session status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "preview-secret") || strings.Contains(rr.Body.String(), "worker-token") {
		t.Fatalf("session leaked credentials: %s", rr.Body.String())
	}
	var result struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil || result.Data.URL != "/worker-preview/preview" {
		t.Fatalf("invalid response: %s", rr.Body.String())
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies=%v", cookies)
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/worker-preview/" {
		t.Fatalf("unsafe cookie: %+v", cookie)
	}
	if a.jwt.ValidateToken(cookie.Value) {
		t.Fatal("preview cookie must not grant access to liao API")
	}
	if time.Until(cookie.Expires) > time.Hour {
		t.Fatal("preview cookie outlives liao session")
	}
	return cookie
}

func workerPreviewTestLogin(t *testing.T, w http.ResponseWriter, r *http.Request, token string) {
	t.Helper()
	if r.Method != http.MethodPost {
		t.Errorf("login method=%s", r.Method)
	}
	if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
		t.Error("forwarded browser credentials to login")
	}
	var credentials map[string]string
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil || credentials["username"] != "preview-user" || credentials["password"] != "preview-secret" {
		t.Errorf("unexpected login credentials")
	}
	w.Header().Set("Set-Cookie", "preview_jwt=upstream-private-token; Path=/; HttpOnly")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"token":%q,"expires_in":3600}`, token)
}

func TestWorkerPreviewEmbedsHTMLAndRewritesMediaWithoutExposingCredentials(t *testing.T) {
	var loginCount atomic.Int32
	var origin string
	a, server := newWorkerPreviewTestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/login" {
			loginCount.Add(1)
			workerPreviewTestLogin(t, w, r, "worker-token")
			return
		}
		if r.Header.Get("Authorization") != "Bearer worker-token" || r.Header.Get("Cookie") != "" {
			t.Error("wrong proxy credentials")
		}
		if r.URL.Query().Has("token") {
			t.Error("browser token leaked upstream")
		}
		switch r.URL.Path {
		case "/preview":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<html><script>fetch('/api/auth/session'); const code="fetch('/api/preview/twitter_is_download')";</script></html>`)
		case "/api/preview/twitter_is_download":
			if r.URL.Query().Get("q") != "中文" {
				t.Error("lost search query")
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"items":[{"id":9007199254740993,"proxy_url":%q,"variants":[{"download_url":"/api/media?url=x&download=1"}],"origin_url":"https://pbs.twimg.com/a.jpg","text":"/api/hello"}]}`, origin+"/api/media?url=https%3A%2F%2Fpbs.twimg.com%2Fa.jpg")
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	origin = server.URL
	cookie := createWorkerPreviewTestSession(t, a)
	for _, endpoint := range []string{"/worker-preview/preview?token=do-not-forward", "/worker-preview/api/preview/twitter_is_download?q=%E4%B8%AD%E6%96%87"} {
		r := httptest.NewRequest(http.MethodGet, endpoint, nil)
		r.AddCookie(cookie)
		r.Header.Set("Authorization", "Bearer browser-token-do-not-forward")
		rr := httptest.NewRecorder()
		a.buildRouter().ServeHTTP(rr, r)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if len(rr.Result().Cookies()) != 0 {
			t.Fatal("upstream cookies reached browser")
		}
		if rr.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("private response cached")
		}
		body := rr.Body.String()
		if strings.Contains(body, "worker-token") || strings.Contains(body, origin) {
			t.Fatalf("upstream credentials/origin leaked: %s", body)
		}
		if strings.HasPrefix(endpoint, "/worker-preview/preview") {
			if !strings.Contains(body, "fetch('/worker-preview/api/auth/session')") || !strings.Contains(body, "/worker-preview/api/preview/twitter_is_download") {
				t.Fatalf("unrewritten html: %s", body)
			}
		} else {
			if !strings.Contains(body, "9007199254740993") || !strings.Contains(body, "/worker-preview/api/media?") || !strings.Contains(body, `"text":"/api/hello"`) || !strings.Contains(body, "https://pbs.twimg.com/a.jpg") {
				t.Fatalf("incorrect JSON rewrite: %s", body)
			}
		}
	}
	if loginCount.Load() != 1 {
		t.Fatalf("logins=%d, want cached login", loginCount.Load())
	}
}

func TestWorkerPreviewRenewsRejectedUpstreamSession(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		t.Run(fmt.Sprint("redirect=", redirect), func(t *testing.T) {
			var logins atomic.Int32
			var requests atomic.Int32
			a, _ := newWorkerPreviewTestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/auth/login" {
					workerPreviewTestLogin(t, w, r, fmt.Sprintf("token-%d", logins.Add(1)))
					return
				}
				requests.Add(1)
				if r.Header.Get("Authorization") == "Bearer token-1" {
					if redirect {
						http.Redirect(w, r, "/login", http.StatusFound)
					} else {
						w.WriteHeader(http.StatusUnauthorized)
					}
					return
				}
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, "PREVIEW_OK")
			})
			cookie := createWorkerPreviewTestSession(t, a)
			r := httptest.NewRequest(http.MethodGet, "/worker-preview/preview", nil)
			r.AddCookie(cookie)
			rr := httptest.NewRecorder()
			a.buildRouter().ServeHTTP(rr, r)
			if rr.Code != http.StatusOK || rr.Body.String() != "PREVIEW_OK" || logins.Load() != 2 || requests.Load() != 2 {
				t.Fatalf("status=%d body=%s logins=%d requests=%d", rr.Code, rr.Body.String(), logins.Load(), requests.Load())
			}
		})
	}
}

func TestWorkerPreviewStreamsRangeAndHead(t *testing.T) {
	a, _ := newWorkerPreviewTestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/login" {
			workerPreviewTestLogin(t, w, r, "worker-token")
			return
		}
		if r.URL.Path != "/api/media" || r.Header.Get("Range") != "bytes=2-4" {
			t.Errorf("unexpected media request: %s headers=%v", r.URL, r.Header)
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Length", "3")
		w.Header().Set("Content-Range", "bytes 2-4/10")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Disposition", "attachment; filename=video.mp4")
		w.Header().Set("Cache-Control", "public, max-age=120")
		w.Header().Set("Set-Cookie", "preview_jwt=hidden")
		w.WriteHeader(http.StatusPartialContent)
		if r.Method != http.MethodHead {
			fmt.Fprint(w, "234")
		}
	})
	cookie := createWorkerPreviewTestSession(t, a)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		r := httptest.NewRequest(method, "/worker-preview/api/media?url=https://video.twimg.com/a.mp4", nil)
		r.AddCookie(cookie)
		r.Header.Set("Range", "bytes=2-4")
		rr := httptest.NewRecorder()
		a.buildRouter().ServeHTTP(rr, r)
		if rr.Code != http.StatusPartialContent || rr.Header().Get("Content-Range") != "bytes 2-4/10" || rr.Header().Get("Content-Disposition") == "" || rr.Header().Get("Cache-Control") != "private, no-store" || rr.Header().Get("Set-Cookie") != "" {
			t.Fatalf("bad media response: status=%d headers=%v", rr.Code, rr.Header())
		}
		if (method == http.MethodGet && rr.Body.String() != "234") || (method == http.MethodHead && rr.Body.Len() != 0) {
			t.Fatalf("bad %s body=%s", method, rr.Body.String())
		}
	}
}

func TestWorkerPreviewRejectsExpiredCookieAndUnknownEndpoints(t *testing.T) {
	var logins atomic.Int32
	a, _ := newWorkerPreviewTestApp(t, func(w http.ResponseWriter, r *http.Request) {
		logins.Add(1)
		workerPreviewTestLogin(t, w, r, "worker-token")
	})
	cookie := createWorkerPreviewTestSession(t, a)
	for _, endpoint := range []struct {
		method, path string
		status       int
	}{
		{http.MethodPost, "/worker-preview/api/auth/login", http.StatusNotFound},
		{http.MethodGet, "/worker-preview/unknown", http.StatusNotFound},
		{http.MethodPost, "/worker-preview/api/media", http.StatusMethodNotAllowed},
	} {
		r := httptest.NewRequest(endpoint.method, endpoint.path, nil)
		r.AddCookie(cookie)
		rr := httptest.NewRecorder()
		a.buildRouter().ServeHTTP(rr, r)
		if rr.Code != endpoint.status {
			t.Fatalf("%s: status=%d", endpoint.path, rr.Code)
		}
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))}).SignedString(a.workerPreview.sessionJWT.secret)
	if err != nil {
		t.Fatal(err)
	}
	cookie.Value = expired
	r := httptest.NewRequest(http.MethodGet, "/worker-preview/preview", nil)
	r.AddCookie(cookie)
	rr := httptest.NewRecorder()
	a.buildRouter().ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized || logins.Load() != 1 {
		t.Fatalf("expired request status=%d, upstream calls=%d", rr.Code, logins.Load())
	}
}

func TestWorkerPreviewDoesNotFollowLoginRedirectOrExposeErrorSecrets(t *testing.T) {
	var redirected atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1) }))
	defer other.Close()
	for _, redirect := range []bool{false, true} {
		a, _ := newWorkerPreviewTestApp(t, func(w http.ResponseWriter, r *http.Request) {
			if redirect {
				http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "preview-secret upstream-private-token")
		})
		token, _ := a.jwt.GenerateToken()
		r := httptest.NewRequest(http.MethodPost, "/api/workerPreview/session", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		a.buildRouter().ServeHTTP(rr, r)
		if rr.Code != http.StatusBadGateway || strings.Contains(rr.Body.String(), "preview-secret") || strings.Contains(rr.Body.String(), "upstream-private-token") || len(rr.Result().Cookies()) != 0 {
			t.Fatalf("unsafe failure: status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
	if redirected.Load() != 0 {
		t.Fatal("credentials followed upstream redirect")
	}
}

func TestWorkerPreviewConcurrentSessionsShareServerLogin(t *testing.T) {
	var logins atomic.Int32
	a, _ := newWorkerPreviewTestApp(t, func(w http.ResponseWriter, r *http.Request) {
		logins.Add(1)
		workerPreviewTestLogin(t, w, r, "worker-token")
	})
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() { defer group.Done(); createWorkerPreviewTestSession(t, a) }()
	}
	group.Wait()
	if logins.Load() != 1 {
		t.Fatalf("logins=%d, want single login", logins.Load())
	}
}

func TestWorkerPreviewLogoutClearsOnlyPreviewCookie(t *testing.T) {
	a := &App{jwt: NewJWTService("test-secret", 1), staticDir: t.TempDir()}
	token, _ := a.jwt.GenerateToken()
	r := httptest.NewRequest(http.MethodDelete, "/api/workerPreview/session", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	a.buildRouter().ServeHTTP(rr, r)
	cookies := rr.Result().Cookies()
	if rr.Code != http.StatusOK || len(cookies) != 1 || cookies[0].Name != workerPreviewCookie || cookies[0].Path != "/worker-preview/" || cookies[0].MaxAge != -1 {
		t.Fatalf("logout status=%d cookies=%v", rr.Code, cookies)
	}
}

func TestWorkerPreviewErrorsAreReadableInEmbeddedPageAndWorkerClient(t *testing.T) {
	a := &App{jwt: NewJWTService("test-secret", 1), staticDir: t.TempDir()}
	for _, endpoint := range []string{"/worker-preview/preview", "/worker-preview/api/auth/session"} {
		rr := httptest.NewRecorder()
		a.buildRouter().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, endpoint, nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d", rr.Code)
		}
		if endpoint == "/worker-preview/preview" {
			if !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/html") || !strings.Contains(rr.Body.String(), "请点击顶部") {
				t.Fatalf("iframe should show readable error, got %s", rr.Body.String())
			}
		} else {
			var result map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil || result["error"] == nil || result["error"] != result["msg"] {
				t.Fatalf("worker client needs error field: %s", rr.Body.String())
			}
		}
	}
}

func TestWorkerPreviewMediaBodyCanOutliveOrdinaryAPITimeout(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	releaseBody := func() { once.Do(func() { close(release) }) }
	defer releaseBody()
	a, server := newWorkerPreviewTestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/login" {
			workerPreviewTestLogin(t, w, r, "worker-token")
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Length", "2")
		fmt.Fprint(w, "a")
		w.(http.Flusher).Flush()
		close(started)
		<-release
		fmt.Fprint(w, "b")
	})
	client := *server.Client()
	client.Timeout = 50 * time.Millisecond
	a.workerPreview = newWorkerPreviewService(a.cfg, &client)
	cookie := createWorkerPreviewTestSession(t, a)
	r := httptest.NewRequest(http.MethodGet, "/worker-preview/api/media?url=https://video.twimg.com/a.mp4", nil)
	r.AddCookie(cookie)
	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { a.buildRouter().ServeHTTP(rr, r); close(done) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("media response did not start")
	}
	// 验证真实超时语义：响应头已收到，响应体传输允许超过普通 API 超时。
	<-time.After(2 * client.Timeout)
	releaseBody()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("media response did not finish")
	}
	if rr.Code != http.StatusOK || rr.Body.String() != "ab" {
		t.Fatalf("media truncated: status=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestWorkerPreviewUnconfiguredShowsActionableError(t *testing.T) {
	a := &App{jwt: NewJWTService("test-secret", 1), staticDir: t.TempDir()}
	token, err := a.jwt.GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/workerPreview/session", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	a.buildRouter().ServeHTTP(rr, r)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s, want 503", rr.Code, rr.Body.String())
	}
}
