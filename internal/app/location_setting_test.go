package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"liao/internal/upstream/wanfeng"
)

var errFakeLocation = errors.New("读取位置设置失败")

// TestWanfengManagerShowCityDefaultPrivate 校验「位置」默认不公开。
func TestWanfengManagerShowCityDefaultPrivate(t *testing.T) {
	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{BaseURL: "https://example.com", Protocol: wanfeng.DefaultProtocol()})
	m := NewWanfengManager(api, wanfeng.NewMemoryBindingStore(), nil)

	// 未注入 provider：按默认值（不公开）
	profile := m.profileFromSign("u1", `{"act":"sign","id":"u1","name":"小明","userSex":"女","userAge":"22"}`)
	if profile.ShowCity == nil || *profile.ShowCity != false {
		t.Fatalf("默认应不公开（ShowCity=false），实际 %+v", profile.ShowCity)
	}
	payload := wanfeng.DefaultProtocol().UpdateProfilePayload(profile)
	if v, ok := payload["showCity"]; !ok || v != false {
		t.Fatalf("update_profile 应带 showCity=false，实际 %+v", payload)
	}
}

// TestWanfengManagerShowCityPerIdentity 校验按身份读取设置，且读库失败时偏向保护（不公开）。
func TestWanfengManagerShowCityPerIdentity(t *testing.T) {
	api := wanfeng.NewAPIClient(wanfeng.APIClientConfig{BaseURL: "https://example.com", Protocol: wanfeng.DefaultProtocol()})
	m := NewWanfengManager(api, wanfeng.NewMemoryBindingStore(), nil)

	settings := map[string]bool{"public-id": true}
	failFor := ""
	m.SetLocationSettingProvider(func(identityID string) (bool, error) {
		if identityID == failFor {
			return true, errFakeLocation
		}
		return settings[identityID], nil
	})

	// 该身份显式设为公开
	profile := m.profileFromSign("public-id", `{"act":"sign","id":"public-id","name":"小明"}`)
	if profile.ShowCity == nil || *profile.ShowCity != true {
		t.Fatalf("public-id 应公开，实际 %+v", profile.ShowCity)
	}

	// 另一个身份没有记录：默认不公开
	profile = m.profileFromSign("other-id", `{"act":"sign","id":"other-id","name":"小红"}`)
	if profile.ShowCity == nil || *profile.ShowCity != false {
		t.Fatalf("未设置的身份应默认不公开，实际 %+v", profile.ShowCity)
	}

	// 读取失败：宁可不公开
	failFor = "broken-id"
	profile = m.profileFromSign("broken-id", `{"act":"sign","id":"broken-id","name":"甲"}`)
	if profile.ShowCity == nil || *profile.ShowCity != false {
		t.Fatalf("读取失败时应偏向不公开，实际 %+v", profile.ShowCity)
	}
}

// TestLocationSettingDefaults 校验位置设置的默认值与空服务容错。
func TestLocationSettingDefaults(t *testing.T) {
	setting := DefaultLocationSetting("u1")
	if setting.IdentityID != "u1" || setting.ShowCity != false {
		t.Fatalf("默认设置应为不公开，实际 %+v", setting)
	}

	// nil 服务（未接数据库）读写都不应报错/panic
	var svc *LocationSettingService
	got, err := svc.Get(nil, "u1")
	if err != nil || got.ShowCity != false {
		t.Fatalf("空服务应返回默认值，实际 %+v err=%v", got, err)
	}
	if err := svc.Save(nil, LocationSetting{IdentityID: "u1", ShowCity: true}); err != nil {
		t.Fatalf("空服务保存应静默成功，实际 %v", err)
	}
	if err := NewLocationSettingService(nil).Save(nil, LocationSetting{}); err == nil {
		t.Fatal("缺少身份ID 时应报错")
	}
}

// TestHandleGetLocationSettingDefaultPrivate 校验默认返回「不公开」且未启用晚风时标记不支持。
func TestHandleGetLocationSettingDefaultPrivate(t *testing.T) {
	a := &App{locationSetting: NewLocationSettingService(nil)}
	req := httptest.NewRequest(http.MethodGet, "/api/getLocationSetting?identityId=IDENT-1", nil)
	rec := httptest.NewRecorder()

	a.handleGetLocationSetting(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"showCity":false`) {
		t.Fatalf("默认应为不公开: %s", body)
	}
	if !strings.Contains(body, `"supported":false`) {
		t.Fatalf("未启用晚风时应标记不支持: %s", body)
	}
}

// TestHandleUpdateLocationSetting 校验保存接口可用（无数据库时降级不报错）。
func TestHandleUpdateLocationSetting(t *testing.T) {
	a := &App{locationSetting: NewLocationSettingService(nil)}

	form := url.Values{}
	form.Set("identityId", "IDENT-1")
	form.Set("showCity", "1")
	req := httptest.NewRequest(http.MethodPost, "/api/updateLocationSetting", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	a.handleUpdateLocationSetting(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d（body=%s）", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"showCity":true`) {
		t.Fatalf("响应未回显设置: %s", rec.Body.String())
	}

	// 缺身份 → 400
	rec = httptest.NewRecorder()
	a.handleUpdateLocationSetting(rec, httptest.NewRequest(http.MethodPost, "/api/updateLocationSetting", strings.NewReader("")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("缺少身份应返回 400，实际 %d", rec.Code)
	}
}
