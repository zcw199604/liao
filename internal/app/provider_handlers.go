package app

import "net/http"

// ProviderInfo 描述一个可选的上游线路。
type ProviderInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     bool   `json:"default"`
}

// handleProviderList 返回当前启用的上游线路列表（前端用于线路选择）。
func (a *App) handleProviderList(w http.ResponseWriter, r *http.Request) {
	providers := []ProviderInfo{
		{
			ID:          ProviderChat2019,
			Name:        "经典线路",
			Description: "原有线路，用户量大、聊天记录完整",
			Default:     true,
		},
	}
	if a.cfg.WanfengEnabled && a.wanfeng != nil {
		providers = append(providers, ProviderInfo{
			ID:          ProviderWanfeng,
			Name:        "晚风线路",
			Description: "新增线路，匹配池不同，支持文字/图片/视频",
			Default:     false,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"data": map[string]any{
			"providers": providers,
		},
	})
}
