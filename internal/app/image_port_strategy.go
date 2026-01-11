package app

// 图片端口策略：按系统配置（fixed/probe/real）解析图片端口；视频端口保持固定逻辑由前端控制。

import (
	"context"
	"strings"
)

func (a *App) getSystemConfigOrDefault(ctx context.Context) SystemConfig {
	if a == nil || a.systemConfig == nil {
		return defaultSystemConfig
	}
	cfg, err := a.systemConfig.Get(ctx)
	if err != nil {
		return defaultSystemConfig
	}
	return cfg
}

func (a *App) getImageServerHostOnly() string {
	if a == nil || a.imageServer == nil {
		return ""
	}
	host := strings.TrimSpace(a.imageServer.GetImgServerHost())
	if host == "" {
		return ""
	}
	return strings.Split(host, ":")[0]
}

func (a *App) resolveImagePortByConfig(ctx context.Context, uploadPath string) string {
	cfg := a.getSystemConfigOrDefault(ctx)
	if cfg.ImagePortFixed == "" {
		cfg.ImagePortFixed = defaultSystemConfig.ImagePortFixed
	}

	imgHost := a.getImageServerHostOnly()
	if imgHost == "" {
		return cfg.ImagePortFixed
	}

	switch cfg.ImagePortMode {
	case ImagePortModeProbe:
		return detectAvailablePort(imgHost)
	case ImagePortModeReal:
		if a != nil && a.imagePortResolver != nil {
			if port, ok := a.imagePortResolver.ResolveByRealRequest(ctx, imgHost, uploadPath, cfg.ImagePortRealMinBytes); ok && port != "" {
				return port
			}
			if cached := a.imagePortResolver.GetCached(imgHost); cached != "" {
				return cached
			}
		}
		// 兜底：如果真实探测失败（如 404/超时/HTML 占位），仍返回一个“可用端口”让前端快速得到响应（展示裂图而不是长时间挂起）。
		if port := strings.TrimSpace(detectAvailablePort(imgHost)); port != "" {
			return port
		}
		return cfg.ImagePortFixed
	case ImagePortModeFixed:
		fallthrough
	default:
		return cfg.ImagePortFixed
	}
}
