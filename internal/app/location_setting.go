package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"liao/internal/database"
)

// 位置（城市）公开设置。
//
// 背景：晚风站点按访客 IP 归属地给身份打一个城市，匹配成功时以 match_result.partner.city
// 下发给对面，落到我方前端就是 sel_userAddress。对方站点设置页的「城市：公开 / 不公开」
// 对应用户资料字段 showCity（对方默认公开）。
//
// 我方策略：**默认不公开**（DefaultShowCity=false），位置在身份设置里由用户自己改。
// 存储按身份落库，晚风线路在登录后通过 update_profile 增量下发 {showCity: <值>}；
// chat2019 线路的前端登录帧本来就把 address_show 写死为 "false"（同样是不公开）。
const (
	// DefaultShowCity 是「位置是否公开」的默认值：不公开。
	DefaultShowCity = false
)

// LocationSetting 是某个身份的位置公开设置。
type LocationSetting struct {
	IdentityID string `json:"identityId"`
	// ShowCity 是否公开位置/城市（false = 不公开，匹配成功时对方看不到）。
	ShowCity bool `json:"showCity"`
}

// DefaultLocationSetting 返回默认设置（不公开）。
func DefaultLocationSetting(identityID string) LocationSetting {
	return LocationSetting{IdentityID: strings.TrimSpace(identityID), ShowCity: DefaultShowCity}
}

// LocationSettingProvider 供 WanfengManager 注入的读取回调（返回 showCity）。
type LocationSettingProvider func(identityID string) (bool, error)

// LocationSettingService 持久化身份位置公开设置。
type LocationSettingService struct {
	db *database.DB
}

// NewLocationSettingService 创建位置设置服务。
func NewLocationSettingService(db *database.DB) *LocationSettingService {
	return &LocationSettingService{db: db}
}

// Get 读取设置；不存在时返回默认值（不公开），不报错。
func (s *LocationSettingService) Get(ctx context.Context, identityID string) (LocationSetting, error) {
	identityID = strings.TrimSpace(identityID)
	if identityID == "" {
		return DefaultLocationSetting(""), nil
	}
	if s == nil || s.db == nil {
		return DefaultLocationSetting(identityID), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	row := s.db.QueryRowContext(ctx,
		`SELECT show_city FROM identity_location_setting WHERE identity_id = ?`, identityID)

	var showCity sql.NullInt64
	if err := row.Scan(&showCity); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DefaultLocationSetting(identityID), nil
		}
		return DefaultLocationSetting(identityID), fmt.Errorf("查询位置设置失败: %w", err)
	}
	return LocationSetting{IdentityID: identityID, ShowCity: showCity.Int64 != 0}, nil
}

// Save 写入（或更新）设置。
func (s *LocationSettingService) Save(ctx context.Context, setting LocationSetting) error {
	setting.IdentityID = strings.TrimSpace(setting.IdentityID)
	if setting.IdentityID == "" {
		return fmt.Errorf("缺少身份ID")
	}
	if s == nil || s.db == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	now := time.Now()
	showCity := 0
	if setting.ShowCity {
		showCity = 1
	}

	var query string
	if s.db.Dialect().Name() == "postgres" {
		query = `INSERT INTO identity_location_setting (identity_id, show_city, created_at, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (identity_id) DO UPDATE SET
				show_city = EXCLUDED.show_city,
				updated_at = EXCLUDED.updated_at`
	} else {
		query = `INSERT INTO identity_location_setting (identity_id, show_city, created_at, updated_at)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				show_city = VALUES(show_city),
				updated_at = VALUES(updated_at)`
	}
	if _, err := s.db.ExecContext(ctx, query, setting.IdentityID, showCity, now, now); err != nil {
		return fmt.Errorf("保存位置设置失败: %w", err)
	}
	return nil
}

// —— HTTP 处理器 ——

// handleGetLocationSetting 返回某个身份的位置公开设置。
func (a *App) handleGetLocationSetting(w http.ResponseWriter, r *http.Request) {
	identityID := strings.TrimSpace(r.URL.Query().Get("identityId"))
	if identityID == "" {
		identityID = strings.TrimSpace(r.FormValue("identityId"))
	}
	if identityID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "缺少身份ID"})
		return
	}

	setting := DefaultLocationSetting(identityID)
	if a.locationSetting != nil {
		if got, err := a.locationSetting.Get(r.Context(), identityID); err == nil {
			setting = got
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0,
		"data": setting,
		// 该身份所在线路是否能真正生效（目前只有晚风线路会下发到上游）
		"supported": a.locationSettingSupported(identityID),
	})
}

// handleUpdateLocationSetting 保存位置公开设置，并在晚风线路上立即下发生效。
func (a *App) handleUpdateLocationSetting(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	identityID := strings.TrimSpace(defaultString(r.FormValue("identityId"), r.URL.Query().Get("identityId")))
	if identityID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": -1, "msg": "缺少身份ID"})
		return
	}

	setting := LocationSetting{
		IdentityID: identityID,
		ShowCity:   parseBoolLoose(r.FormValue("showCity")),
	}
	if a.locationSetting != nil {
		if err := a.locationSetting.Save(r.Context(), setting); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"code": -1, "msg": "保存失败: " + err.Error()})
			return
		}
	}

	// 晚风线路：已连接时立即下发；未连接时下次登录后会自动带上
	if a.wanfeng != nil && a.wanfeng.Supported() {
		if err := a.wanfeng.ApplyShowCity(identityID, setting.ShowCity); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"code": 0,
				"data": setting,
				"msg":  "已保存，将在下次连接时生效",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "data": setting})
}

// locationSettingSupported 判断该身份所在线路是否支持「位置公开」设置。
//
// chat2019 线路不在我方控制范围内（前端登录帧写死 address_show=false，等同不公开），
// 因此只有绑定过晚风的身份返回 true。
func (a *App) locationSettingSupported(identityID string) bool {
	if a == nil || a.wanfeng == nil || !a.wanfeng.Supported() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, ok := a.wanfengIdentityUID(ctx, identityID)
	return ok
}
