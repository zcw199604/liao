package app

// 系统配置与图片端口解析 API（JWT 保护，所有用户共用）。

import (
	"encoding/json"
	"io"
	"net/http"
)

func (a *App) handleGetSystemConfig(w http.ResponseWriter, r *http.Request) {
	cfg := a.getSystemConfigOrDefault(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": cfg,
	})
}

type updateSystemConfigRequest struct {
	ImagePortMode         *string `json:"imagePortMode"`
	ImagePortFixed        *string `json:"imagePortFixed"`
	ImagePortRealMinBytes *int64  `json:"imagePortRealMinBytes"`
}

func (a *App) handleUpdateSystemConfig(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.systemConfig == nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": "系统配置服务未初始化"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": "读取请求失败"})
		return
	}

	var req updateSystemConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": "JSON解析失败"})
		return
	}

	current := a.getSystemConfigOrDefault(r.Context())
	next := current
	if req.ImagePortMode != nil {
		next.ImagePortMode = ImagePortMode(*req.ImagePortMode)
	}
	if req.ImagePortFixed != nil {
		next.ImagePortFixed = *req.ImagePortFixed
	}
	if req.ImagePortRealMinBytes != nil {
		next.ImagePortRealMinBytes = *req.ImagePortRealMinBytes
	}

	updated, err := a.systemConfig.Update(r.Context(), next)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": err.Error()})
		return
	}

	if a.imagePortResolver != nil {
		a.imagePortResolver.ClearAll()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": updated,
	})
}

type resolveImagePortRequest struct {
	Path string `json:"path"`
}

func (a *App) handleResolveImagePort(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": "读取请求失败"})
		return
	}

	var req resolveImagePortRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": -1, "msg": "JSON解析失败"})
		return
	}

	port := a.resolveImagePortByConfig(r.Context(), req.Path)
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": map[string]any{
			"port": port,
		},
	})
}
