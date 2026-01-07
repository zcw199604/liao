package app

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
)

func (a *App) handleRecordImageSend(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	remoteURL := r.FormValue("remoteUrl")
	fromUserID := r.FormValue("fromUserId")
	toUserID := r.FormValue("toUserId")
	localFilename := r.FormValue("localFilename")

	record, err := a.mediaUpload.RecordImageSend(r.Context(), remoteURL, fromUserID, toUserID, localFilename)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "记录失败: " + err.Error(),
		})
		return
	}

	if record == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"message": "未找到原始上传记录",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "记录成功",
		"data":    record,
	})
}

func (a *App) handleGetUserUploadHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	pageSize := parseIntDefault(r.URL.Query().Get("pageSize"), 20)
	hostHeader := requestHostHeader(r)

	list, err := a.mediaUpload.GetUserUploadHistory(r.Context(), userID, page, pageSize, hostHeader)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "查询失败: " + err.Error(),
		})
		return
	}

	total, err := a.mediaUpload.GetUserUploadCount(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "查询失败: " + err.Error(),
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "查询成功",
		"data": map[string]any{
			"list":       list,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

func (a *App) handleGetUserSentImages(w http.ResponseWriter, r *http.Request) {
	fromUserID := r.URL.Query().Get("fromUserId")
	toUserID := r.URL.Query().Get("toUserId")
	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	pageSize := parseIntDefault(r.URL.Query().Get("pageSize"), 20)
	hostHeader := requestHostHeader(r)

	list, err := a.mediaUpload.GetUserSentImages(r.Context(), fromUserID, toUserID, page, pageSize, hostHeader)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "查询失败: " + err.Error(),
		})
		return
	}

	total, err := a.mediaUpload.GetUserSentCount(r.Context(), fromUserID, toUserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "查询失败: " + err.Error(),
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "查询成功",
		"data": map[string]any{
			"list":       list,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

func (a *App) handleGetUserUploadStats(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")

	total, err := a.mediaUpload.GetUserUploadCount(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "查询失败: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "查询成功",
		"data": map[string]any{
			"totalCount": total,
		},
	})
}

func (a *App) handleGetChatImages(w http.ResponseWriter, r *http.Request) {
	userID1 := r.URL.Query().Get("userId1")
	userID2 := r.URL.Query().Get("userId2")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 20)
	hostHeader := requestHostHeader(r)

	images, err := a.mediaUpload.GetChatImages(r.Context(), userID1, userID2, limit, hostHeader)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, []string{})
		return
	}
	writeJSON(w, http.StatusOK, images)
}

func (a *App) handleReuploadHistoryImage(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	userID := r.FormValue("userId")
	localPath := r.FormValue("localPath")
	cookieData := defaultString(r.FormValue("cookieData"), "")
	referer := defaultString(r.FormValue("referer"), "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj")
	userAgent := defaultString(r.FormValue("userAgent"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := a.mediaUpload.ReuploadLocalFile(r.Context(), userID, localPath, cookieData, referer, userAgent)
	if err != nil {
		writeText(w, http.StatusInternalServerError, "{\"state\":\"ERROR\",\"msg\":\""+err.Error()+"\"}")
		return
	}

	if parseJSONStateOK(resp) {
		a.imageCache.AddImageToCache(userID, localPath)
	}

	writeText(w, http.StatusOK, resp)
}

func (a *App) handleGetAllUploadImages(w http.ResponseWriter, r *http.Request) {
	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	pageSize := parseIntDefault(r.URL.Query().Get("pageSize"), 20)
	hostHeader := requestHostHeader(r)

	list, err := a.mediaUpload.GetAllUploadImagesWithDetails(r.Context(), page, pageSize, hostHeader)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	total, err := a.mediaUpload.GetAllUploadImagesCount(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	imgServerHost := strings.Split(a.imageServer.GetImgServerHost(), ":")[0]
	availablePort := detectAvailablePort(imgServerHost)

	writeJSON(w, http.StatusOK, map[string]any{
		"port":       availablePort,
		"data":       list,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": int(math.Ceil(float64(total) / float64(pageSize))),
	})
}

func (a *App) handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	localPath := r.FormValue("localPath")
	userID := r.FormValue("userId")

	result, err := a.mediaUpload.DeleteMediaByPath(r.Context(), userID, localPath)
	if err != nil {
		// 兼容 Java：RuntimeException -> 403，其它异常 -> 500
		if errors.Is(err, ErrDeleteForbidden) || strings.Contains(err.Error(), "无权") {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"code": 403,
				"msg":  err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code": 500,
			"msg":  "删除失败：" + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "删除成功",
		"data": map[string]any{
			"deletedRecords": result.DeletedRecords,
			"fileDeleted":    result.FileDeleted,
		},
	})
}

func (a *App) handleBatchDeleteMedia(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     string   `json:"userId"`
		LocalPaths []string `json:"localPaths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code": 400,
			"msg":  "文件路径列表不能为空",
		})
		return
	}

	if strings.TrimSpace(req.UserID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "用户ID不能为空"})
		return
	}
	if len(req.LocalPaths) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "文件路径列表不能为空"})
		return
	}
	if len(req.LocalPaths) > 50 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "单次最多删除50张图片"})
		return
	}

	result, err := a.mediaUpload.BatchDeleteMedia(r.Context(), req.UserID, req.LocalPaths)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code": 500,
			"msg":  "批量删除失败：" + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "批量删除完成",
		"data": result,
	})
}

func parseIntDefault(raw string, def int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return n
	}
	return def
}
