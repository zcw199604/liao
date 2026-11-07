package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"liao/internal/database"
	"liao/internal/upstream/wanfeng"
)

// 三期：匹配偏好（期望对方性别/年龄区间）。
//
// 晚风站点把偏好挂在用户资料上，只能通过 update_profile 下发；因此我方必须本地
// 保存一份，登录帧与 update_profile 才能带上。chat2019 线路没有该概念，接口对
// 它返回默认值（不生效、也不报错），避免前端需要判断线路。

const (
	// PreferGenderAny 表示不限性别。
	PreferGenderAny = "any"
)

// MatchPreferenceProvider 读取某个身份的匹配偏好（供 WanfengManager 注入，避免反向依赖）。
type MatchPreferenceProvider func(identityID string) (MatchPreference, error)

// MatchPreference 是某个身份的匹配偏好。
type MatchPreference struct {
	IdentityID         string `json:"identityId"`
	PreferGender       string `json:"preferGender"`       // male/female/any
	PreferAgeMin       int    `json:"preferAgeMin"`       // 0 表示不限
	PreferAgeMax       int    `json:"preferAgeMax"`       // 0 表示不限
	PreferSameProvince bool   `json:"preferSameProvince"` // 是否只匹配同省
}

// DefaultMatchPreference 返回默认偏好（不限）。
func DefaultMatchPreference(identityID string) MatchPreference {
	return MatchPreference{IdentityID: identityID, PreferGender: PreferGenderAny}
}

// Normalize 归一化偏好，避免脏值下发到对方站点。
func (p MatchPreference) Normalize() MatchPreference {
	out := p
	out.IdentityID = strings.TrimSpace(out.IdentityID)
	switch strings.ToLower(strings.TrimSpace(out.PreferGender)) {
	case "male", "m", "男":
		out.PreferGender = "male"
	case "female", "f", "女":
		out.PreferGender = "female"
	default:
		out.PreferGender = PreferGenderAny
	}
	if out.PreferAgeMin < 0 {
		out.PreferAgeMin = 0
	}
	if out.PreferAgeMax < 0 {
		out.PreferAgeMax = 0
	}
	// 区间颠倒时自动纠正，避免对方站点拒绝
	if out.PreferAgeMax > 0 && out.PreferAgeMin > out.PreferAgeMax {
		out.PreferAgeMin, out.PreferAgeMax = out.PreferAgeMax, out.PreferAgeMin
	}
	return out
}

// UpstreamPayload 转成对方 update_profile 的 matchPreference 字段。
//
// 字段名沿用对方站点（preferGender/preferAgeMin/preferAgeMax/preferSameProvince）。
func (p MatchPreference) UpstreamPayload() map[string]any {
	payload := map[string]any{
		"preferGender": p.PreferGender,
	}
	if p.PreferAgeMin > 0 {
		payload["preferSenderAgeMin"] = p.PreferAgeMin
	}
	if p.PreferAgeMax > 0 {
		payload["preferSenderAgeMax"] = p.PreferAgeMax
	}
	if p.PreferSameProvince {
		payload["preferSameProvince"] = true
	}
	return payload
}

// MatchPreferenceService 持久化身份匹配偏好。
type MatchPreferenceService struct {
	db *database.DB
}

// NewMatchPreferenceService 创建偏好服务。
func NewMatchPreferenceService(db *database.DB) *MatchPreferenceService {
	return &MatchPreferenceService{db: db}
}

// Get 读取偏好；不存在时返回默认值（不报错）。
func (s *MatchPreferenceService) Get(ctx context.Context, identityID string) (MatchPreference, error) {
	identityID = strings.TrimSpace(identityID)
	if identityID == "" {
		return DefaultMatchPreference(""), nil
	}
	if s == nil || s.db == nil {
		return DefaultMatchPreference(identityID), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	row := s.db.QueryRowContext(ctx, `SELECT prefer_gender, prefer_age_min, prefer_age_max, prefer_same_province
		FROM identity_match_preference WHERE identity_id = ?`, identityID)

	var (
		gender   sql.NullString
		ageMin   sql.NullInt64
		ageMax   sql.NullInt64
		sameProv sql.NullInt64
	)
	if err := row.Scan(&gender, &ageMin, &ageMax, &sameProv); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DefaultMatchPreference(identityID), nil
		}
		return DefaultMatchPreference(identityID), fmt.Errorf("查询匹配偏好失败: %w", err)
	}

	pref := MatchPreference{
		IdentityID:         identityID,
		PreferGender:       gender.String,
		PreferAgeMin:       int(ageMin.Int64),
		PreferAgeMax:       int(ageMax.Int64),
		PreferSameProvince: sameProv.Int64 != 0,
	}
	return pref.Normalize(), nil
}

// Save 写入（或更新）偏好。
func (s *MatchPreferenceService) Save(ctx context.Context, pref MatchPreference) error {
	pref = pref.Normalize()
	if pref.IdentityID == "" {
		return fmt.Errorf("缺少身份ID")
	}
	if s == nil || s.db == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	now := time.Now()
	sameProv := 0
	if pref.PreferSameProvince {
		sameProv = 1
	}

	query := `INSERT INTO identity_match_preference (
		identity_id, prefer_gender, prefer_age_min, prefer_age_max, prefer_same_province, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		pref.IdentityID, pref.PreferGender, pref.PreferAgeMin, pref.PreferAgeMax,
		sameProv, now, now,
	}

	if s.db.Dialect().Name() == "postgres" {
		query = `INSERT INTO identity_match_preference (
			identity_id, prefer_gender, prefer_age_min, prefer_age_max, prefer_same_province, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (identity_id) DO UPDATE SET
			prefer_gender = EXCLUDED.prefer_gender,
			prefer_age_min = EXCLUDED.prefer_age_min,
			prefer_age_max = EXCLUDED.prefer_age_max,
			prefer_same_province = EXCLUDED.prefer_same_province,
			updated_at = EXCLUDED.updated_at`
	} else {
		query = `INSERT INTO identity_match_preference (
			identity_id, prefer_gender, prefer_age_min, prefer_age_max, prefer_same_province, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			prefer_gender = VALUES(prefer_gender),
			prefer_age_min = VALUES(prefer_age_min),
			prefer_age_max = VALUES(prefer_age_max),
			prefer_same_province = VALUES(prefer_same_province),
			updated_at = VALUES(updated_at)`
	}

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("保存匹配偏好失败: %w", err)
	}
	return nil
}

// —— HTTP 处理器 ——

// handleGetMatchPreference 返回某个身份的匹配偏好。
func (a *App) handleGetMatchPreference(w http.ResponseWriter, r *http.Request) {
	identityID := strings.TrimSpace(r.URL.Query().Get("identityId"))
	if identityID == "" {
		identityID = strings.TrimSpace(r.FormValue("identityId"))
	}
	if identityID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "缺少身份ID"})
		return
	}

	pref, err := a.matchPreference.Get(r.Context(), identityID)
	if err != nil {
		// 读取失败降级为默认值：前端只用来渲染表单，不应整块报错
		pref = DefaultMatchPreference(identityID)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"data": pref,
		// 该线路是否真正支持匹配偏好（chat2019 不支持，前端据此隐藏入口）
		"supported": a.matchPreferenceSupported(requestProviderContext(r), identityID),
	})
}

// handleUpdateMatchPreference 保存匹配偏好，并在晚风线路上立即下发生效。
func (a *App) handleUpdateMatchPreference(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	identityID := strings.TrimSpace(defaultString(r.FormValue("identityId"), r.URL.Query().Get("identityId")))
	if identityID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "缺少身份ID"})
		return
	}

	pref := MatchPreference{
		IdentityID:         identityID,
		PreferGender:       r.FormValue("preferGender"),
		PreferAgeMin:       atoiTrim(r.FormValue("preferAgeMin")),
		PreferAgeMax:       atoiTrim(r.FormValue("preferAgeMax")),
		PreferSameProvince: parseBoolLoose(r.FormValue("preferSameProvince")),
	}.Normalize()

	if err := a.matchPreference.Save(r.Context(), pref); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "保存失败: " + err.Error()})
		return
	}

	// 晚风线路：立刻下发 update_profile，否则要等下次重连才生效
	if a.wanfeng != nil && a.wanfeng.Supported() {
		if _, ok := a.wanfengIdentityUID(requestProviderContext(r), identityID); ok {
			profile, _ := a.wanfeng.ProfileOf(identityID)
			profile.MatchPreference = pref.UpstreamPayload()
			if err := a.wanfeng.SendAction(identityID, a.wanfeng.Protocol().ActionUpdateProfile,
				a.wanfeng.Protocol().UpdateProfilePayload(profile)); err != nil {
				// 连接没就绪不算失败：下次登录帧会带上偏好
				writeJSON(w, http.StatusOK, map[string]any{
					"code": 0,
					"data": pref,
					"msg":  "偏好已保存，将在下次连接时生效",
				})
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "data": pref})
}

// matchPreferenceSupported 判断该身份所在线路是否支持匹配偏好。
func (a *App) matchPreferenceSupported(baseCtx context.Context, identityID string) bool {
	if a == nil || a.wanfeng == nil || !a.wanfeng.Supported() {
		return false
	}
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	// 继承请求 context（可能已带「当前线路」），再加 3 秒超时。
	ctx, cancel := context.WithTimeout(baseCtx, 3*time.Second)
	defer cancel()
	_, ok := a.wanfengIdentityUID(ctx, identityID)
	return ok
}

// ApplyMatchPreferenceToProfile 把本地偏好塞进要上送的展示资料。
//
// 登录帧与 update_profile 都要带上，否则对方站点会按默认偏好匹配。
func (a *App) ApplyMatchPreferenceToProfile(ctx context.Context, identityID string, profile wanfeng.Profile) wanfeng.Profile {
	if a == nil || a.matchPreference == nil {
		return profile
	}
	pref, err := a.matchPreference.Get(ctx, identityID)
	if err != nil {
		return profile
	}
	// 全默认值时不带，避免覆盖对方侧已有的偏好
	if pref.PreferGender == PreferGenderAny && pref.PreferAgeMin == 0 &&
		pref.PreferAgeMax == 0 && !pref.PreferSameProvince {
		return profile
	}
	profile.MatchPreference = pref.UpstreamPayload()
	return profile
}

// atoiTrim 宽松解析整数（空/非法返回 0）。
func atoiTrim(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}

// parseBoolLoose 宽松解析布尔（前端可能传 true/1/on）。
func parseBoolLoose(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}
