package app

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

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

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL, strings.NewReader(form.Encode()))
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
