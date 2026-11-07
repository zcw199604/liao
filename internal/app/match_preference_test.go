package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"liao/internal/upstream/wanfeng"
)

// TestMatchPreferenceNormalize 校验偏好归一化（脏值不下发到对方站点）。
func TestMatchPreferenceNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   MatchPreference
		want MatchPreference
	}{
		{
			name: "中文性别",
			in:   MatchPreference{IdentityID: "u1", PreferGender: "女"},
			want: MatchPreference{IdentityID: "u1", PreferGender: "female"},
		},
		{
			name: "未知性别回落到不限",
			in:   MatchPreference{IdentityID: "u1", PreferGender: "外星人"},
			want: MatchPreference{IdentityID: "u1", PreferGender: PreferGenderAny},
		},
		{
			name: "区间颠倒自动纠正",
			in:   MatchPreference{IdentityID: "u1", PreferGender: "male", PreferAgeMin: 40, PreferAgeMax: 20},
			want: MatchPreference{IdentityID: "u1", PreferGender: "male", PreferAgeMin: 20, PreferAgeMax: 40},
		},
		{
			name: "负数归零",
			in:   MatchPreference{IdentityID: "u1", PreferGender: "male", PreferAgeMin: -5, PreferAgeMax: -1},
			want: MatchPreference{IdentityID: "u1", PreferGender: "male"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Normalize()
			if got != tc.want {
				t.Fatalf("归一化结果不符:\n got=%+v\nwant=%+v", got, tc.want)
			}
		})
	}
}

// TestMatchPreferenceUpstreamPayload 校验转成对方 matchPreference 字段。
func TestMatchPreferenceUpstreamPayload(t *testing.T) {
	pref := MatchPreference{
		IdentityID:         "u1",
		PreferGender:       "female",
		PreferAgeMin:       22,
		PreferAgeMax:       30,
		PreferSameProvince: true,
	}
	payload := pref.UpstreamPayload()
	if payload["preferGender"] != "female" {
		t.Fatalf("性别字段不符: %+v", payload)
	}
	if payload["preferSenderAgeMin"] != 22 || payload["preferSenderAgeMax"] != 30 {
		t.Fatalf("年龄字段不符（对方用 preferSenderAge*）: %+v", payload)
	}
	if payload["preferSameProvince"] != true {
		t.Fatalf("同省字段不符: %+v", payload)
	}

	// 0 表示不限，不应带上（否则对方会把 0 当成真实年龄下限）
	bare := MatchPreference{IdentityID: "u1", PreferGender: PreferGenderAny}.UpstreamPayload()
	if _, ok := bare["preferSenderAgeMin"]; ok {
		t.Fatalf("不限年龄不应带 preferSenderAgeMin: %+v", bare)
	}
}

// TestHandleUpdateMatchPreferenceWithoutDB 校验无数据库时接口仍可用（降级不报错）。
func TestHandleUpdateMatchPreferenceWithoutDB(t *testing.T) {
	a := &App{matchPreference: NewMatchPreferenceService(nil)}

	form := url.Values{}
	form.Set("identityId", "IDENT-1")
	form.Set("preferGender", "female")
	form.Set("preferAgeMin", "22")
	form.Set("preferAgeMax", "30")
	form.Set("preferSameProvince", "1")

	req := httptest.NewRequest(http.MethodPost, "/api/updateMatchPreference", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	a.handleUpdateMatchPreference(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d（body=%s）", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"preferGender":"female"`) {
		t.Fatalf("响应未回显偏好: %s", rec.Body.String())
	}
}

// TestHandleUpdateMatchPreferenceRequiresIdentity 校验缺身份时返回 400。
func TestHandleUpdateMatchPreferenceRequiresIdentity(t *testing.T) {
	a := &App{matchPreference: NewMatchPreferenceService(nil)}
	req := httptest.NewRequest(http.MethodPost, "/api/updateMatchPreference", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	a.handleUpdateMatchPreference(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("缺少身份应返回 400，实际 %d", rec.Code)
	}
}

// TestHandleGetMatchPreferenceReportsUnsupported 校验 chat2019 身份返回 supported=false。
func TestHandleGetMatchPreferenceReportsUnsupported(t *testing.T) {
	a := &App{matchPreference: NewMatchPreferenceService(nil)}
	req := httptest.NewRequest(http.MethodGet, "/api/getMatchPreference?identityId=IDENT-1", nil)
	rec := httptest.NewRecorder()

	a.handleGetMatchPreference(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"supported":false`) {
		t.Fatalf("未启用晚风时应标记不支持: %s", rec.Body.String())
	}
}

// TestManagerCarriesMatchPreferenceIntoLogin 校验登录帧会带上本地匹配偏好。
func TestManagerCarriesMatchPreferenceIntoLogin(t *testing.T) {
	m := newTestWanfengManager(t, nil)
	m.SetMatchPreferenceProvider(func(identityID string) (MatchPreference, error) {
		if identityID != "IDENT-1" {
			return DefaultMatchPreference(identityID), nil
		}
		return MatchPreference{
			IdentityID:   "IDENT-1",
			PreferGender: "female",
			PreferAgeMin: 22,
			PreferAgeMax: 30,
		}, nil
	})

	profile := m.applyMatchPreference("IDENT-1", wanfeng.Profile{Nickname: "我", Gender: "male", Age: 25})
	if profile.MatchPreference == nil {
		t.Fatal("偏好应并入展示资料")
	}
	if profile.MatchPreference["preferGender"] != "female" {
		t.Fatalf("偏好内容不符: %+v", profile.MatchPreference)
	}

	// 登录帧 payload 必须带上
	payload := wanfeng.DefaultProtocol().LoginPayload(
		wanfeng.Binding{UpstreamUID: "WFU1", DeviceID: "d1"}, profile, false)
	pref, ok := payload["matchPreference"]
	if !ok {
		t.Fatalf("登录帧应带匹配偏好: %+v", payload)
	}
	_ = pref

	// 全默认偏好时不应带
	bare := m.applyMatchPreference("IDENT-2", wanfeng.Profile{Nickname: "我"})
	if bare.MatchPreference != nil {
		t.Fatalf("默认偏好不应携带: %+v", bare.MatchPreference)
	}
}

// TestLoginPayloadIncludesMatchPreference 直接校验协议层把偏好写进登录帧。
func TestLoginPayloadIncludesMatchPreference(t *testing.T) {
	proto := wanfeng.DefaultProtocol()
	payload := proto.LoginPayload(wanfeng.Binding{UpstreamUID: "WFU1"}, wanfeng.Profile{
		Nickname:        "我",
		Gender:          "male",
		Age:             24,
		MatchPreference: map[string]any{"preferGender": "female"},
	}, false)

	pref, ok := payload["matchPreference"].(map[string]any)
	if !ok || pref["preferGender"] != "female" {
		t.Fatalf("登录帧未携带偏好: %+v", payload)
	}
}

// TestApplyMatchPreferenceToProfile 校验 App 侧读取偏好并注入资料。
func TestApplyMatchPreferenceToProfile(t *testing.T) {
	a := &App{matchPreference: NewMatchPreferenceService(nil)}
	// 无数据库时 Get 返回默认值，注入后不应带偏好
	profile := a.ApplyMatchPreferenceToProfile(context.Background(), "IDENT-1", wanfeng.Profile{Nickname: "我"})
	if profile.MatchPreference != nil {
		t.Fatalf("默认偏好不应注入: %+v", profile.MatchPreference)
	}

	// 服务为 nil 时原样返回
	empty := &App{}
	got := empty.ApplyMatchPreferenceToProfile(context.Background(), "IDENT-1", wanfeng.Profile{Nickname: "我"})
	if got.Nickname != "我" {
		t.Fatalf("资料应原样返回: %+v", got)
	}
}
