package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"liao/internal/upstream/wanfeng"
)

// TestMediaFilenameFor 校验转存文件名生成（对方 URL 可能没有扩展名）。
func TestMediaFilenameFor(t *testing.T) {
	cases := []struct {
		fileID      string
		kind        string
		contentType string
		remoteURL   string
		want        string
	}{
		{"abc123", wanfeng.MediaKindImage, "image/jpeg", "https://x/files/abc123", "abc123.jpg"},
		{"abc123.jpg", wanfeng.MediaKindImage, "", "https://x/files/abc123.jpg", "abc123.jpg"},
		{"abc123", wanfeng.MediaKindVideo, "", "https://x/files/abc123", "abc123.mp4"},
		{"abc123", wanfeng.MediaKindAudio, "audio/mp4", "", "abc123.m4a"},
		{"abc123", wanfeng.MediaKindImage, "image/png; charset=binary", "", "abc123.png"},
		// 无 fileId 时用时间戳兜底，但仍需有扩展名
		{"", wanfeng.MediaKindVideo, "", "", ".mp4"},
	}
	for _, c := range cases {
		got := mediaFilenameFor(c.fileID, c.kind, c.contentType, c.remoteURL)
		if c.fileID == "" {
			if !strings.HasSuffix(got, ".mp4") {
				t.Fatalf("兜底文件名应有扩展名，实际 %q", got)
			}
			continue
		}
		if got != c.want {
			t.Fatalf("mediaFilenameFor(%q,%q,%q,%q)=%q，期望 %q", c.fileID, c.kind, c.contentType, c.remoteURL, got, c.want)
		}
	}
}

// TestExtensionFromContentType 校验 Content-Type → 扩展名。
func TestExtensionFromContentType(t *testing.T) {
	cases := map[string]string{
		"image/jpeg":               ".jpg",
		"IMAGE/PNG":                ".png",
		"image/webp; charset=x":    ".webp",
		"video/mp4":                ".mp4",
		"audio/mp4":                ".m4a",
		"application/octet-stream": "",
		"":                         "",
	}
	for ct, want := range cases {
		if got := extensionFromContentType(ct); got != want {
			t.Fatalf("extensionFromContentType(%q)=%q，期望 %q", ct, got, want)
		}
	}
}

// TestContentTypeFromFilename 校验文件名 → Content-Type。
func TestContentTypeFromFilename(t *testing.T) {
	cases := map[string]string{
		"a.JPG":  "image/jpeg",
		"a.webp": "image/webp",
		"a.mp4":  "video/mp4",
		"a.m4a":  "audio/mp4",
		"a.zzz":  "application/octet-stream",
	}
	for name, want := range cases {
		if got := contentTypeFromFilename(name); got != want {
			t.Fatalf("contentTypeFromFilename(%q)=%q，期望 %q", name, got, want)
		}
	}
}

// TestWanfengMediaBridgeCache 校验双向缓存命中，避免重复搬运。
func TestWanfengMediaBridgeCache(t *testing.T) {
	b := &wanfengMediaBridge{
		outward: make(map[string]mediaCacheEntry),
		inward:  make(map[string]mediaCacheEntry),
	}

	b.put(b.outward, "2026/11/a.jpg", "https://x/files/a.jpg")
	if got, ok := b.get(b.outward, "2026/11/a.jpg"); !ok || got != "https://x/files/a.jpg" {
		t.Fatalf("缓存读取失败: %q ok=%v", got, ok)
	}
	if _, ok := b.get(b.outward, "missing"); ok {
		t.Fatal("不存在的键不应命中")
	}

	// 过期后应失效
	b.outward["old"] = mediaCacheEntry{value: "v", expires: zeroTime()}
	if _, ok := b.get(b.outward, "old"); ok {
		t.Fatal("过期条目不应命中")
	}
	if _, exists := b.outward["old"]; exists {
		t.Fatal("过期条目应被清理")
	}
}

// TestWanfengMediaBridgeBucket 校验按媒体类型选择上传桶。
func TestWanfengMediaBridgeBucket(t *testing.T) {
	proto := wanfeng.DefaultProtocol()
	b := &wanfengMediaBridge{proto: proto}
	if got := b.bucketFor(wanfeng.MediaKindAudio); got != proto.BucketAudio {
		t.Fatalf("语音应使用语音桶，实际 %q", got)
	}
	if got := b.bucketFor(wanfeng.MediaKindImage); got != proto.BucketImage {
		t.Fatalf("图片应使用图片桶，实际 %q", got)
	}
	if got := b.bucketFor(wanfeng.MediaKindVideo); got != proto.BucketImage {
		t.Fatalf("视频应使用图片桶，实际 %q", got)
	}
}

// TestBuildImageServerCookie 校验合成 Cookie 的格式（对方按 id_昵称_时间戳_随机 解析）。
func TestBuildImageServerCookie(t *testing.T) {
	cookie := buildImageServerCookie("user1", "小明")
	parts := strings.Split(cookie, "_")
	if len(parts) < 4 || parts[0] != "user1" || parts[1] != "小明" {
		t.Fatalf("Cookie 格式不符: %q", cookie)
	}
	// 空值兜底
	cookie = buildImageServerCookie("", "")
	if !strings.HasPrefix(cookie, "0_") {
		t.Fatalf("空用户应兜底为 0，实际 %q", cookie)
	}
}

// TestFetchBytesViaServer 用本地 HTTP 服务校验字节搬运。
func TestFetchBytesViaServer(t *testing.T) {
	payload := []byte("hello-media")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	b := &wanfengMediaBridge{app: &App{httpClient: srv.Client()}}
	data, contentType, err := b.fetchBytes(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("搬运失败: %v", err)
	}
	if string(data) != string(payload) || contentType != "image/jpeg" {
		t.Fatalf("搬运结果不符: data=%q ct=%q", data, contentType)
	}

	// 非 2xx 应报错
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer errSrv.Close()
	if _, _, err := b.fetchBytes(context.Background(), errSrv.URL); err == nil {
		t.Fatal("404 应返回错误")
	}

	// 空内容应报错
	emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = io.WriteString(w, "")
	}))
	defer emptySrv.Close()
	if _, _, err := b.fetchBytes(context.Background(), emptySrv.URL); err == nil {
		t.Fatal("空内容应返回错误")
	}
}

// zeroTime 返回一个已过期的时间，便于测试缓存失效。
func zeroTime() time.Time { return time.Now().Add(-time.Hour) }

// TestUploadToImageServer 用本地服务模拟 chat2019 媒体服务器，校验上传与响应解析。
func TestUploadToImageServer(t *testing.T) {
	var gotCookie, gotFilename string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			t.Errorf("表单解析失败: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("upload_file")
		if err != nil {
			t.Errorf("缺少文件字段: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		gotFilename = header.Filename
		_, _ = io.Copy(io.Discard, file)
		_, _ = io.WriteString(w, `{"state":"OK","msg":"2026/11/07/Random/fake.png","code":1}`)
	}))
	defer srv.Close()

	b := &wanfengMediaBridge{
		app: &App{
			imageServer: imageServerFromURL(t, srv.URL),
			mediaUpload: &MediaUploadService{httpClient: srv.Client()},
			httpClient:  srv.Client(),
		},
		outward: make(map[string]mediaCacheEntry),
		inward:  make(map[string]mediaCacheEntry),
	}

	remoteFilename, err := b.uploadToImageServer(context.Background(), "user1", "pic.png", []byte("bytes"))
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	if remoteFilename != "2026/11/07/Random/fake.png" {
		t.Fatalf("应返回远程文件名，实际 %q", remoteFilename)
	}
	if gotFilename != "pic.png" {
		t.Fatalf("上传文件名不符: %q", gotFilename)
	}
	if !strings.HasPrefix(gotCookie, "user1_") {
		t.Fatalf("Cookie 应带用户 id，实际 %q", gotCookie)
	}
}

// TestMirrorInboundEndToEnd 校验「对方 URL → 下载 → 转存到我方媒体服务器」全链路。
func TestMirrorInboundEndToEnd(t *testing.T) {
	payload := []byte("remote-image-bytes")

	// 对方媒体服务（带查询参数，模拟签名 URL）
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(payload)
	}))
	defer remote.Close()

	// 我方媒体服务器
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("upload_file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != string(payload) {
			t.Errorf("转存内容不一致: %q", data)
		}
		if !strings.HasPrefix(header.Filename, "abc123") || !strings.HasSuffix(header.Filename, ".png") {
			t.Errorf("转存文件名应保留 fileId 与扩展名，实际 %q", header.Filename)
		}
		_, _ = io.WriteString(w, `{"state":"OK","msg":"2026/11/07/Random/mirrored.png"}`)
	}))
	defer local.Close()

	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{
		BaseURL:    remote.URL,
		Protocol:   wanfeng.DefaultProtocol(),
		HTTPClient: remote.Client(),
	})
	b := newWanfengMediaBridge(&App{
		imageServer: imageServerFromURL(t, local.URL),
		mediaUpload: &MediaUploadService{httpClient: local.Client()},
		httpClient:  local.Client(),
	}, api, nil)

	name, err := b.MirrorInbound("user1", wanfeng.MediaKindImage, remote.URL+"/files/abc123.png?token=xyz")
	if err != nil {
		t.Fatalf("转存失败: %v", err)
	}
	if name != "2026/11/07/Random/mirrored.png" {
		t.Fatalf("应返回我方媒体文件名，实际 %q", name)
	}

	// 二次调用应命中缓存（不再请求任何服务）
	again, err := b.MirrorInbound("user1", wanfeng.MediaKindImage, remote.URL+"/files/abc123.png?token=other")
	if err != nil || again != name {
		t.Fatalf("应命中缓存，实际 %q err=%v", again, err)
	}
}

// imageServerFromURL 按 httptest 地址构造媒体服务器（GetImgServerHost 返回 host:port）。
func imageServerFromURL(t *testing.T, rawURL string) *ImageServerService {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("解析测试地址失败: %v", err)
	}
	svc := NewImageServerService(parsed.Hostname(), parsed.Port())
	if got := svc.GetImgServerHost(); got != parsed.Host {
		t.Fatalf("媒体服务器地址不符: got=%q want=%q", got, parsed.Host)
	}
	return svc
}

// TestMirrorOutboundEndToEnd 校验「我方文件名 → 拉取字节 → 上传到对方」全链路。
func TestMirrorOutboundEndToEnd(t *testing.T) {
	payload := []byte("local-image-bytes")

	// 我方媒体服务器：提供 /img/Upload/<文件名>
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/img/Upload/2026/11/07/Random/local.png" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(payload)
	}))
	defer local.Close()

	// 对方上传接口：校验字段与内容
	var gotUser, gotBucket string
	var gotBytes []byte
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wanfeng.DefaultProtocol().UploadPath {
			t.Errorf("上传路径不符: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		gotUser = r.FormValue(wanfeng.DefaultProtocol().UploadUserField)
		gotBucket = r.FormValue(wanfeng.DefaultProtocol().UploadBucketField)
		file, _, err := r.FormFile(wanfeng.DefaultProtocol().UploadFileField)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		gotBytes, _ = io.ReadAll(file)
		_, _ = io.WriteString(w, `{"code":200,"success":true,"data":{"filePath":"/files/remote.png"}}`)
	}))
	defer remote.Close()

	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{
		BaseURL:    remote.URL,
		Protocol:   wanfeng.DefaultProtocol(),
		HTTPClient: remote.Client(),
	})
	b := newWanfengMediaBridge(&App{
		imageServer: imageServerFromURL(t, local.URL),
		mediaUpload: &MediaUploadService{httpClient: local.Client()},
		httpClient:  local.Client(),
	}, api, nil)

	gotURL, err := b.MirrorOutbound("user1", wanfeng.MediaKindImage, "2026/11/07/Random/local.png")
	if err != nil {
		t.Fatalf("上传到对方失败: %v", err)
	}
	if gotURL != remote.URL+"/files/remote.png" {
		t.Fatalf("应返回对方绝对 URL，实际 %q", gotURL)
	}
	if string(gotBytes) != string(payload) {
		t.Fatalf("上传内容不一致: %q", gotBytes)
	}
	if gotBucket != wanfeng.DefaultProtocol().BucketImage {
		t.Fatalf("桶名不符: %q", gotBucket)
	}
	_ = gotUser

	// 二次调用命中缓存
	again, err := b.MirrorOutbound("user1", wanfeng.MediaKindImage, "2026/11/07/Random/local.png")
	if err != nil || again != gotURL {
		t.Fatalf("应命中缓存，实际 %q err=%v", again, err)
	}
	// 双向登记：对方 URL 能反查回文件名（自己回显复用）
	if name, ok := b.get(b.inward, wanfeng.MediaURLKey(gotURL)); !ok || name != "2026/11/07/Random/local.png" {
		t.Fatalf("应登记反向映射，实际 %q ok=%v", name, ok)
	}
}

// TestWanfengMediaBridgeImplementsInterface 编译期断言：app 层实现 wanfeng.MediaBridge。
var _ wanfeng.MediaBridge = (*wanfengMediaBridge)(nil)

// TestWanfengManagerPropagatesMediaBridge 校验配置媒体桥后，新建与已存在的客户端都能拿到。
func TestWanfengManagerPropagatesMediaBridge(t *testing.T) {
	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{BaseURL: "https://example.com", Protocol: wanfeng.DefaultProtocol()})
	m := NewWanfengManager(api, wanfeng.NewMemoryBindingStore(), nil)

	bridge := &wanfengMediaBridge{
		outward: make(map[string]mediaCacheEntry),
		inward:  make(map[string]mediaCacheEntry),
	}
	m.SetMediaBridge(bridge)

	m.mu.Lock()
	got := m.media
	m.mu.Unlock()
	if got == nil {
		t.Fatal("管理器应保存媒体桥")
	}

	// 模拟「先建客户端、后配媒体桥」的场景
	m.mu.Lock()
	m.clients["u1"] = wanfeng.NewClient(wanfeng.ClientConfig{
		IdentityID: "u1",
		API:        api,
		Bindings:   wanfeng.NewMemoryBindingStore(),
		Emitter:    wanfeng.Emitter(nopEmitter{}),
	})
	client := m.clients["u1"]
	m.mu.Unlock()

	// 已存在的客户端也应被刷新
	if client.MediaBridge() != nil {
		t.Fatal("新建客户端的媒体桥此时应为空")
	}
	m.SetMediaBridge(bridge)
	if client.MediaBridge() != bridge {
		t.Fatal("已存在的客户端应被更新媒体桥")
	}
}

// nopEmitter 丢弃所有下发（测试用）。
type nopEmitter struct{}

func (nopEmitter) Emit(string, map[string]any)      {}
func (nopEmitter) EmitFatal(string, map[string]any) {}

// TestClientForIsolatesByIdentity 校验不同身份使用独立 Cookie Jar 的客户端。
func TestClientForIsolatesByIdentity(t *testing.T) {
	store := wanfeng.NewMemoryBindingStore()
	ctx := context.Background()
	if err := store.Save(ctx, wanfeng.Binding{
		IdentityID: "u1", UpstreamUID: "WFU1", IdentityToken: "v1.t1", DeviceID: "d1",
	}); err != nil {
		t.Fatalf("保存绑定失败: %v", err)
	}

	base := wanfeng.NewAPIClient(wanfeng.APIClientConfig{BaseURL: "https://example.com", Protocol: wanfeng.DefaultProtocol()})
	m := NewWanfengManager(base, store, nil)
	b := newWanfengMediaBridge(&App{wanfeng: m}, base, nil)

	first, err := b.clientFor(ctx, "u1")
	if err != nil {
		t.Fatalf("获取客户端失败: %v", err)
	}
	if first == base {
		t.Fatal("已绑定的身份应使用独立客户端")
	}
	again, err := b.clientFor(ctx, "u1")
	if err != nil || again != first {
		t.Fatalf("同身份应复用客户端: %v", err)
	}

	// 未绑定的身份退回共享客户端
	if fallback, err := b.clientFor(ctx, "unknown"); err != nil || fallback != base {
		t.Fatalf("未绑定身份应退回共享客户端: %v", err)
	}
}

// TestParseReceiveMediaUsesPayloadField 校验对方 receive_message 的 payload 字段能被识别。
func TestParseReceiveMediaUsesPayloadField(t *testing.T) {
	proto := wanfeng.DefaultProtocol()
	info, ok := proto.ParseReceiveMedia(map[string]any{
		"type":    "image",
		"payload": "/files/2026/11/07/abc.jpg?token=1",
		"fromUid": "WAT1",
		"tid":     "local_1_2",
	})
	if !ok || info.Kind != wanfeng.MediaKindImage || info.URL == "" || info.FromUID != "WAT1" {
		t.Fatalf("payload 字段解析失败: %+v ok=%v", info, ok)
	}
}
