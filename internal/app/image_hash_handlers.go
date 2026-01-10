package app

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleCheckDuplicateMedia 上传文件并在 image_hash 表中查重：
// 1) md5_hash 精确命中：直接返回命中列表
// 2) md5 未命中：尝试计算 pHash，并返回相似度>=阈值的结果（含 similarity/distance）
// 说明：仅对可解码图片计算 pHash；视频/不可解码格式仅做 MD5 查重。
func (a *App) handleCheckDuplicateMedia(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if err := r.ParseMultipartForm(110 << 20); err != nil {
		slog.Error("查重请求解析失败", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "请求解析失败"})
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": "缺少文件"})
		return
	}
	fileHeader := files[0]

	limit := parseIntOrDefault(r.FormValue("limit"), 20)
	limit = clampInt(limit, 1, 200)

	thresholdType, similarityThreshold, distanceThreshold, err := resolvePHashThreshold(
		r.FormValue("similarityThreshold"),
		r.FormValue("distanceThreshold"),
		r.FormValue("threshold"),
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": 400, "msg": err.Error()})
		return
	}

	md5Value, err := a.fileStorage.CalculateMD5(fileHeader)
	if err != nil {
		slog.Error("查重MD5计算失败", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "MD5计算失败"})
		return
	}

	md5Matches, err := a.imageHash.FindByMD5Hash(r.Context(), md5Value, limit)
	if err != nil {
		slog.Error("查重MD5查询失败", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "查询失败"})
		return
	}
	if len(md5Matches) > 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{
				"matchType":           "md5",
				"md5":                 md5Value,
				"thresholdType":       thresholdType,
				"similarityThreshold": similarityThreshold,
				"distanceThreshold":   distanceThreshold,
				"limit":               limit,
				"items":               md5Matches,
			},
		})
		slog.Info("查重完成(md5命中)", "fileName", fileHeader.Filename, "md5", md5Value, "count", len(md5Matches), "totalMs", time.Since(start).Milliseconds())
		return
	}

	phash, err := a.imageHash.CalculatePHash(fileHeader)
	if err != nil {
		// 不可计算 pHash（视频/非图片/解码失败等）：按“无命中”返回
		writeJSON(w, http.StatusOK, map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{
				"matchType":           "none",
				"md5":                 md5Value,
				"thresholdType":       thresholdType,
				"similarityThreshold": similarityThreshold,
				"distanceThreshold":   distanceThreshold,
				"limit":               limit,
				"reason":              ErrPHashUnsupported.Error(),
				"items":               []ImageHashMatch{},
			},
		})
		slog.Info("查重完成(无md5且无法计算phash)", "fileName", fileHeader.Filename, "md5", md5Value, "totalMs", time.Since(start).Milliseconds())
		return
	}

	phashMatches, err := a.imageHash.FindSimilarByPHash(r.Context(), phash, distanceThreshold, limit)
	if err != nil {
		slog.Error("查重pHash查询失败", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": 500, "msg": "查询失败"})
		return
	}

	matchType := "none"
	if len(phashMatches) > 0 {
		matchType = "phash"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"msg":  "success",
		"data": map[string]any{
			"matchType":           matchType,
			"md5":                 md5Value,
			"pHash":               strconv.FormatInt(phash, 10),
			"thresholdType":       thresholdType,
			"similarityThreshold": similarityThreshold,
			"distanceThreshold":   distanceThreshold,
			"limit":               limit,
			"items":               phashMatches,
		},
	})

	slog.Info(
		"查重完成(phash)",
		"fileName", fileHeader.Filename,
		"md5", md5Value,
		"phash", phash,
		"thresholdType", thresholdType,
		"similarityThreshold", similarityThreshold,
		"distanceThreshold", distanceThreshold,
		"count", len(phashMatches),
		"totalMs", time.Since(start).Milliseconds(),
	)
}

func parseSimilarityThreshold(raw string, defaultValue float64) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return clampFloat(defaultValue, 0, 1), nil
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}

	// 兼容 0-100 输入：大于 1 视为百分比
	if v > 1 {
		v = v / 100
	}
	return clampFloat(v, 0, 1), nil
}

func resolvePHashThreshold(similarityRaw, distanceRaw, thresholdRaw string) (thresholdType string, similarityThreshold float64, distanceThreshold int, err error) {
	// 兼容：优先 similarityThreshold（0-1 或 0-100），否则使用 distanceThreshold/threshold（汉明距离，0-64）。
	similarityRaw = strings.TrimSpace(similarityRaw)
	distanceRaw = strings.TrimSpace(distanceRaw)
	thresholdRaw = strings.TrimSpace(thresholdRaw)

	if similarityRaw != "" {
		thresholdType = "similarity"
		similarityThreshold, err = parseSimilarityThreshold(similarityRaw, 0.0)
		if err != nil {
			return "", 0, 0, fmt.Errorf("similarityThreshold 参数非法")
		}
		distanceThreshold = similarityThresholdToDistance(similarityThreshold)
		return thresholdType, similarityThreshold, distanceThreshold, nil
	}

	thresholdType = "distance"
	raw := distanceRaw
	if raw == "" {
		raw = thresholdRaw
	}
	distanceThreshold = parseIntOrDefault(raw, 10)
	distanceThreshold = clampInt(distanceThreshold, 0, phashBitLength)
	similarityThreshold = similarityFromDistance(distanceThreshold)
	return thresholdType, similarityThreshold, distanceThreshold, nil
}

func similarityThresholdToDistance(threshold float64) int {
	threshold = clampFloat(threshold, 0, 1)
	maxDist := int(math.Floor((1-threshold)*phashBitLength + 1e-9))
	return clampInt(maxDist, 0, phashBitLength)
}

func parseIntOrDefault(raw string, defaultValue int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return v
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
