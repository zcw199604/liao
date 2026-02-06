package cookiecloud

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient_ValidationErrors(t *testing.T) {
	if _, err := NewClient(" "); err == nil {
		t.Fatalf("want error for empty baseURL")
	}
	if _, err := NewClient("example.com/api"); err == nil {
		t.Fatalf("want error for missing scheme")
	}
}

func TestFindCookiesForDomain_AndBuildHeader(t *testing.T) {
	cookieData := map[string][]Cookie{
		".douyin.com": {
			{Name: "a", Value: "1"},
			{Name: "", Value: "skip"},
		},
	}

	cookies := FindCookiesForDomain(cookieData, "douyin.com")
	if len(cookies) != 2 {
		t.Fatalf("cookies=%v", cookies)
	}
	if got := BuildCookieHeader(cookies); got != "a=1" {
		t.Fatalf("header=%q", got)
	}
	if got := BuildCookieHeader(nil); got != "" {
		t.Fatalf("header nil=%q", got)
	}
	if got := FindCookiesForDomain(cookieData, " "); got != nil {
		t.Fatalf("domain empty should return nil")
	}
	if got := FindCookiesForDomain(cookieData, ".douyin.com"); len(got) != 2 {
		t.Fatalf("leading dot domain got=%v", got)
	}
}

func TestDecrypt_ErrorPaths(t *testing.T) {
	if _, err := Decrypt("u", "$$$", "p", CryptoTypeAES128CBCFixed); err == nil {
		t.Fatalf("want base64 decode error")
	}

	short := base64.StdEncoding.EncodeToString([]byte("abc"))
	if _, err := Decrypt("u", short, "p", CryptoTypeAES128CBCFixed); err == nil {
		t.Fatalf("want invalid ciphertext length error")
	}

	if _, err := Decrypt("u", short, "p", CryptoTypeLegacy); err == nil {
		t.Fatalf("want legacy payload too short error")
	}

	invalidLegacy := base64.StdEncoding.EncodeToString([]byte("12345678" + "12345678"))
	if _, err := Decrypt("u", invalidLegacy, "p", CryptoTypeLegacy); err == nil {
		t.Fatalf("want legacy prefix error")
	}

	if _, err := pkcs7Unpad([]byte{}, 16); err == nil {
		t.Fatalf("want empty data error")
	}
	if _, err := pkcs7Unpad([]byte{1, 2, 3}, 16); err == nil {
		t.Fatalf("want not aligned error")
	}
}

func TestClient_GetEncryptedAndCookieHeaderErrors(t *testing.T) {
	t.Run("empty uuid", func(t *testing.T) {
		c, err := NewClient("https://example.com/api")
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		if _, err := c.GetEncrypted(context.Background(), "", ""); err == nil {
			t.Fatalf("want empty uuid error")
		}
	})

	t.Run("http error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusBadGateway)
		}))
		defer srv.Close()

		c, _ := NewClient(srv.URL)
		_, err := c.GetEncrypted(context.Background(), "u1", "")
		if err == nil || !strings.Contains(err.Error(), "http 502") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("decode error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not-json"))
		}))
		defer srv.Close()

		c, _ := NewClient(srv.URL)
		_, err := c.GetEncrypted(context.Background(), "u1", "")
		if err == nil || !strings.Contains(err.Error(), "decode response") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("missing encrypted", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"crypto_type":"aes-128-cbc-fixed"}`))
		}))
		defer srv.Close()

		c, _ := NewClient(srv.URL)
		_, err := c.GetEncrypted(context.Background(), "u1", "")
		if err == nil || !strings.Contains(err.Error(), "missing encrypted") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("no cookies for domain", func(t *testing.T) {
		uuid := "u1"
		password := "p1"
		plain := DecryptedData{
			CookieData:       map[string][]Cookie{".example.com": {{Name: "a", Value: "1"}}},
			LocalStorageData: map[string]map[string]string{},
			UpdateTime:       "2026-02-01T00:00:00Z",
		}
		plainBytes, err := json.Marshal(plain)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		encrypted := encryptFixedForMoreTest(t, uuid, password, plainBytes)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"encrypted":"` + encrypted + `","crypto_type":"aes-128-cbc-fixed"}`))
		}))
		defer srv.Close()

		c, _ := NewClient(srv.URL)
		_, err = c.GetCookieHeader(context.Background(), uuid, password, "douyin.com", CryptoTypeAES128CBCFixed)
		if err == nil || !strings.Contains(err.Error(), "no cookies for domain") {
			t.Fatalf("err=%v", err)
		}
	})
}

func encryptFixedForMoreTest(t *testing.T, uuid, password string, plain []byte) string {
	t.Helper()
	return encryptFixed(t, uuid, password, plain)
}
