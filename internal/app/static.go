package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var filepathRelFn = filepath.Rel

func (a *App) spaHandler() http.Handler {
	indexPath := filepath.Join(a.staticDir, "index.html")
	fileServer := http.FileServer(http.Dir(a.staticDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		// 尝试按静态文件返回（若不存在则交给 SPA 回退判定）
		clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		fullPath := filepath.Join(a.staticDir, clean)
		if fi, err := os.Stat(fullPath); err == nil && !fi.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA 回退：支持 createWebHistory() 刷新（/list、/chat/... 等）
		// - 浏览器刷新/直达页面通常带 Accept: text/html
		// - 无扩展名路径也视为前端路由（避免尾随 / 或新增路由漏配导致 404）
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/html") || filepath.Ext(r.URL.Path) == "" {
			http.ServeFile(w, r, indexPath)
			return
		}

		http.NotFound(w, r)
	})
}

func (a *App) uploadFileServer() http.Handler {
	uploadFS := http.StripPrefix("/upload", http.FileServer(http.Dir("upload")))

	baseTempAbs := ""
	if a != nil && a.fileStorage != nil {
		baseTempAbs = strings.TrimSpace(a.fileStorage.baseTempAbs)
	}
	if baseTempAbs == "" {
		baseTempAbs = filepath.Join(os.TempDir(), "video_extract_inputs")
	}
	tempFS := http.StripPrefix("/upload/"+tempVideoExtractInputsDir, http.FileServer(http.Dir(baseTempAbs)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && strings.HasPrefix(r.URL.Path, "/upload/"+tempVideoExtractInputsDir+"/") {
			tempFS.ServeHTTP(w, r)
			return
		}
		uploadFS.ServeHTTP(w, r)
	})
}

func (a *App) lspFileServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("lsp文件请求", "rawPath", r.URL.Path, "requestURI", r.RequestURI)

		// 将 URL 路径 /lsp/... 映射到本地根目录（默认 /lsp，可通过 LSP_ROOT 配置）。
		// 安全要求：禁止 path traversal（..），确保最终路径始终在根目录内。
		fullPath, fi, err := resolveExistingLspLocalPath(a.cfg.LspRoot, r.URL.Path)
		if err != nil {
			slog.Warn("lsp路径非法", "path", r.URL.Path, "error", err)
			http.NotFound(w, r)
			return
		}
		slog.Info("lsp文件路径", "fullPath", fullPath)

		slog.Info("lsp文件找到", "fullPath", fullPath, "size", fi.Size())
		// 提供文件服务
		http.ServeFile(w, r, fullPath)
	})
}

func resolveExistingLspLocalPath(root, requestPath string) (string, os.FileInfo, error) {
	candidates, err := resolveLspLocalPathCandidates(root, requestPath)
	if err != nil {
		return "", nil, err
	}

	var firstPath string
	var firstErr error
	for _, candidate := range candidates {
		if firstPath == "" {
			firstPath = candidate
		}
		fi, err := os.Stat(candidate)
		if err == nil && !fi.IsDir() {
			return candidate, fi, nil
		}
		if firstErr == nil {
			if err != nil {
				firstErr = err
			} else {
				firstErr = fmt.Errorf("路径是目录")
			}
		}
	}

	if firstErr == nil {
		firstErr = os.ErrNotExist
	}
	return firstPath, nil, firstErr
}

func resolveLspLocalPathCandidates(root, requestPath string) ([]string, error) {
	variants := lspRequestPathSpaceVariants(requestPath)
	candidates := make([]string, 0, len(variants))
	seen := map[string]struct{}{}
	var firstErr error

	for _, variant := range variants {
		fullPath, err := resolveLspLocalPath(root, variant)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if _, ok := seen[fullPath]; ok {
			continue
		}
		seen[fullPath] = struct{}{}
		candidates = append(candidates, fullPath)
	}

	if len(candidates) == 0 {
		if firstErr == nil {
			firstErr = fmt.Errorf("路径解析失败")
		}
		return nil, firstErr
	}
	return candidates, nil
}

func lspRequestPathSpaceVariants(requestPath string) []string {
	variants := []string{requestPath}
	seen := map[string]struct{}{requestPath: {}}
	add := func(v string) {
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		variants = append(variants, v)
	}

	if strings.Contains(requestPath, " ") {
		add(strings.ReplaceAll(requestPath, " ", "%20"))
	}
	decodedSpaces := decodeEscapedSpacesOnly(requestPath)
	if decodedSpaces != requestPath {
		add(decodedSpaces)
	}

	return variants
}

func decodeEscapedSpacesOnly(value string) string {
	if !strings.Contains(value, "%") {
		return value
	}

	var b strings.Builder
	b.Grow(len(value))
	changed := false
	for i := 0; i < len(value); {
		if i+2 < len(value) && value[i] == '%' && value[i+1] == '2' && value[i+2] == '0' {
			b.WriteByte(' ')
			i += 3
			changed = true
			continue
		}
		b.WriteByte(value[i])
		i++
	}
	if !changed {
		return value
	}
	return b.String()
}

func resolveLspLocalPath(root, requestPath string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = filepath.Join(string(filepath.Separator), "lsp")
	}
	root = filepath.Clean(root)

	if !strings.HasPrefix(requestPath, "/lsp") {
		return "", fmt.Errorf("不支持的路径前缀")
	}

	rel := strings.TrimPrefix(requestPath, "/lsp")
	rel = strings.TrimPrefix(rel, "/")

	// path.Clean 用于 URL 路径（固定为 / 分隔），避免 Windows 反斜杠导致绕过。
	cleanURL := path.Clean("/" + rel) // 始终以 / 开头
	if cleanURL == "/" {
		return "", fmt.Errorf("不支持目录访问")
	}
	if strings.HasPrefix(cleanURL, "/..") {
		return "", fmt.Errorf("检测到路径遍历")
	}

	// 将 URL 路径转换为系统路径，并拼接到根目录。
	target := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(cleanURL, "/")))

	// 二次校验：确保 target 仍位于 root 之下。
	rel2, err := filepathRelFn(root, target)
	if err != nil {
		return "", fmt.Errorf("路径解析失败: %w", err)
	}
	if rel2 == "." || strings.HasPrefix(rel2, ".."+string(filepath.Separator)) || rel2 == ".." {
		return "", fmt.Errorf("检测到路径越界")
	}

	return target, nil
}
