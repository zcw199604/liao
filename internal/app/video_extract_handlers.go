package app

// 视频抽帧 API：提供探测、创建任务、查询任务、取消、继续与删除等接口。
// 说明：接口遵循项目 camelCase 路径约定，并复用既有 JWT 中间件进行鉴权。

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func (a *App) handleUploadVideoExtractInput(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.fileStorage == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "服务未初始化"})
		return
	}

	if err := r.ParseMultipartForm(220 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "请求解析失败: " + err.Error()})
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "缺少文件"})
		return
	}
	fh := files[0]

	// 说明：该接口用于“抽帧任务输入”，前端文件选择不做类型限制；
	// 服务端仅负责落盘并返回 localPath，后续由 /api/probeVideo 使用 ffprobe 校验是否为可解析的视频。
	localPath, err := a.fileStorage.SaveFile(fh, "video")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "保存失败: " + err.Error()})
		return
	}

	hostHeader := requestHostHeader(r)
	url := ""
	if strings.TrimSpace(hostHeader) != "" {
		url = "http://" + strings.TrimSpace(hostHeader) + "/upload" + localPath
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": map[string]any{
			"localPath":         localPath,
			"url":               url,
			"originalFilename":  fh.Filename,
			"localFilename":     filepath.Base(strings.TrimPrefix(localPath, "/")),
			"fileSize":          fh.Size,
			"contentType":       strings.TrimSpace(fh.Header.Get("Content-Type")),
			"suggestSourceType": string(VideoExtractSourceUpload),
		},
	})
}

func (a *App) handleProbeVideo(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.videoExtract == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "服务未初始化"})
		return
	}

	sourceType := VideoExtractSourceType(strings.TrimSpace(r.URL.Query().Get("sourceType")))
	localPath := strings.TrimSpace(r.URL.Query().Get("localPath"))
	md5Value := strings.TrimSpace(r.URL.Query().Get("md5"))

	abs := ""
	switch sourceType {
	case VideoExtractSourceUpload:
		lp := normalizeUploadLocalPathInput(localPath)
		if lp == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "localPath 不能为空"})
			return
		}
		path, err := a.videoExtract.resolveUploadAbsPath(lp)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": err.Error()})
			return
		}
		abs = path
	case VideoExtractSourceMtPhoto:
		if md5Value == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "md5 不能为空"})
			return
		}
		if !isHexMD5(md5Value) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "md5 非法"})
			return
		}
		if a.mtPhoto == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "mtPhoto 未配置"})
			return
		}
		item, err := a.mtPhoto.ResolveFilePath(r.Context(), md5Value)
		if err != nil || item == nil {
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "解析 mtPhoto 文件路径失败: " + err.Error()})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "解析 mtPhoto 文件路径失败"})
			return
		}
		full, err := resolveLspLocalPath(a.cfg.LspRoot, item.FilePath)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "文件路径非法: " + err.Error()})
			return
		}
		abs = full
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "sourceType 非法"})
		return
	}

	probe, err := a.videoExtract.ProbeVideo(r.Context(), abs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": probe,
	})
}

func (a *App) handleCreateVideoExtractTask(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.videoExtract == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "服务未初始化"})
		return
	}

	var req VideoExtractCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "请求解析失败"})
		return
	}

	taskID, probe, err := a.videoExtract.CreateTask(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": map[string]any{
			"taskId": taskID,
			"probe":  probe,
		},
	})
}

func (a *App) handleGetVideoExtractTaskList(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.videoExtract == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "服务未初始化"})
		return
	}

	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	pageSize := parseIntDefault(r.URL.Query().Get("pageSize"), 20)
	hostHeader := requestHostHeader(r)

	items, total, err := a.videoExtract.ListTasks(r.Context(), page, pageSize, hostHeader)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"data": map[string]any{
			"items":    items,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

func (a *App) handleGetVideoExtractTaskDetail(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.videoExtract == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "服务未初始化"})
		return
	}

	taskID := strings.TrimSpace(r.URL.Query().Get("taskId"))
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "taskId 不能为空"})
		return
	}

	cursor := parseIntDefault(r.URL.Query().Get("cursor"), 0)
	pageSize := parseIntDefault(r.URL.Query().Get("pageSize"), a.cfg.VideoExtractFramePageSz)
	if pageSize <= 0 {
		pageSize = a.cfg.VideoExtractFramePageSz
	}

	hostHeader := requestHostHeader(r)
	task, frames, err := a.videoExtract.GetTaskDetail(r.Context(), taskID, cursor, pageSize, hostHeader)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"data": map[string]any{
			"task":   task,
			"frames": frames,
		},
	})
}

func (a *App) handleCancelVideoExtractTask(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.videoExtract == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "服务未初始化"})
		return
	}
	var req VideoExtractCancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "请求解析失败"})
		return
	}
	if err := a.videoExtract.CancelAndMark(r.Context(), req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "msg": "success"})
}

func (a *App) handleContinueVideoExtractTask(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.videoExtract == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "服务未初始化"})
		return
	}
	var req VideoExtractContinueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "请求解析失败"})
		return
	}
	if err := a.videoExtract.ContinueTask(r.Context(), req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "msg": "success"})
}

func (a *App) handleDeleteVideoExtractTask(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.videoExtract == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "服务未初始化"})
		return
	}
	var req VideoExtractDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "请求解析失败"})
		return
	}
	if err := a.videoExtract.DeleteTask(r.Context(), req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "msg": "success"})
}

// parseIntDefault 在 media_history_handlers.go 中定义；这里补一个 float 解析兜底避免重复问询。
func parseFloatDefault(raw string, def float64) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return def
	}
	return f
}
