package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"liao/internal/config"
)

func TestHandleRuntimeConfigReturnsRandomVIPCode(t *testing.T) {
	a := &App{cfg: config.Config{RandomVIPCode: "vip-from-env"}}
	req := httptest.NewRequest(http.MethodGet, "/api/runtimeConfig", nil)
	rr := httptest.NewRecorder()

	a.handleRuntimeConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var got struct {
		Code int `json:"code"`
		Data struct {
			RandomVIPCode string `json:"randomVipCode"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Code != 0 || got.Data.RandomVIPCode != "vip-from-env" {
		t.Fatalf("response=%+v", got)
	}
}
