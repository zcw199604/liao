package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var newHTTPRequestWithContextFn = http.NewRequestWithContext

func (a *App) handleDeleteUpstreamUser(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	myUserID := strings.TrimSpace(r.FormValue("myUserId"))
	userToID := strings.TrimSpace(r.FormValue("userToId"))
	if myUserID == "" || userToID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "参数不能为空"})
		return
	}

	resp := map[string]any{}
	upstreamURL := "http://v1.chat2019.cn/asmx/method.asmx/Del_User"

	form := url.Values{}
	form.Set("myUserID", myUserID)
	form.Set("UserToID", userToID)
	form.Set("vipcode", "")
	form.Set("serverPort", "1001")

	req, err := newHTTPRequestWithContextFn(r.Context(), http.MethodPost, upstreamURL, strings.NewReader(form.Encode()))
	if err != nil {
		resp["code"] = -1
		resp["msg"] = "删除失败: " + err.Error()
		writeJSON(w, http.StatusOK, resp)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := a.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	res, err := httpClient.Do(req)
	if err != nil {
		resp["code"] = -1
		resp["msg"] = "删除失败: " + err.Error()
		writeJSON(w, http.StatusOK, resp)
		return
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	resp["code"] = 0
	resp["msg"] = "success"
	resp["data"] = string(body)
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) handleBatchDeleteUpstreamUsers(w http.ResponseWriter, r *http.Request) {
	var in struct {
		MyUserID  string   `json:"myUserId"`
		UserToIDs []string `json:"userToIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "参数解析失败"})
		return
	}

	myUserID := strings.TrimSpace(in.MyUserID)
	if myUserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "myUserId不能为空"})
		return
	}
	if len(in.UserToIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "userToIds不能为空"})
		return
	}

	// 去空、去重，保留首次出现顺序，避免重复请求上游。
	seen := make(map[string]struct{}, len(in.UserToIDs))
	userToIDs := make([]string, 0, len(in.UserToIDs))
	for _, raw := range in.UserToIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		userToIDs = append(userToIDs, id)
	}
	if len(userToIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "userToIds不能为空"})
		return
	}
	if len(userToIDs) > 200 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "单次最多删除200个会话"})
		return
	}

	type batchDeleteResult struct {
		SuccessCount int                 `json:"successCount"`
		FailCount    int                 `json:"failCount"`
		FailedItems  []map[string]string `json:"failedItems"`
	}

	result := batchDeleteResult{
		FailedItems: make([]map[string]string, 0),
	}

	httpClient := a.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	upstreamURL := "http://v1.chat2019.cn/asmx/method.asmx/Del_User"
	for _, userToID := range userToIDs {
		form := url.Values{}
		form.Set("myUserID", myUserID)
		form.Set("UserToID", userToID)
		form.Set("vipcode", "")
		form.Set("serverPort", "1001")

		upstreamReq, err := newHTTPRequestWithContextFn(r.Context(), http.MethodPost, upstreamURL, strings.NewReader(form.Encode()))
		if err != nil {
			result.FailCount++
			result.FailedItems = append(result.FailedItems, map[string]string{
				"userToId": userToID,
				"reason":   "删除失败: " + err.Error(),
			})
			continue
		}
		upstreamReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		res, err := httpClient.Do(upstreamReq)
		if err != nil {
			result.FailCount++
			result.FailedItems = append(result.FailedItems, map[string]string{
				"userToId": userToID,
				"reason":   "删除失败: " + err.Error(),
			})
			continue
		}
		// 尽量读完 body 以复用 keep-alive 连接；上游响应通常很小，这里限制读取大小即可。
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4<<20))
		_ = res.Body.Close()
		result.SuccessCount++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "批量删除完成",
		"data": result,
	})
}

func (a *App) handleGetConnectionStats(w http.ResponseWriter, r *http.Request) {
	if a.wsManager == nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": "WebSocket管理器未初始化"})
		return
	}
	stats := a.wsManager.GetConnectionStats()
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": stats,
	})
}

func (a *App) handleDisconnectAllConnections(w http.ResponseWriter, r *http.Request) {
	if a.wsManager == nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": "WebSocket管理器未初始化"})
		return
	}
	a.wsManager.CloseAllConnections()
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "所有连接已断开",
	})
}

func (a *App) handleGetForceoutUserCount(w http.ResponseWriter, r *http.Request) {
	if a.forceoutManager == nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": "Forceout管理器未初始化"})
		return
	}
	count := a.forceoutManager.GetForbiddenUserCount()
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"data": count,
	})
}

func (a *App) handleClearForceoutUsers(w http.ResponseWriter, r *http.Request) {
	if a.forceoutManager == nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": "Forceout管理器未初始化"})
		return
	}
	count := a.forceoutManager.ClearAllForceout()
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  fmt.Sprintf("已清除%d个被禁止的用户", count),
	})
}
