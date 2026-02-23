package cookiecloud

import (
	"context"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestFindCookiesForDomain_EmptyCookieData(t *testing.T) {
	if got := FindCookiesForDomain(map[string][]Cookie{}, "example.com"); got != nil {
		t.Fatalf("expected nil, got=%v", got)
	}
}

func TestDecrypt_JSONUnmarshalError(t *testing.T) {
	uuid := "u-json"
	password := "p-json"
	encrypted := encryptFixedForMoreTest(t, uuid, password, []byte("not-json"))
	if _, err := Decrypt(uuid, encrypted, password, CryptoTypeAES128CBCFixed); err == nil || !strings.Contains(err.Error(), "json unmarshal") {
		t.Fatalf("expected json unmarshal error, got=%v", err)
	}
}

func TestDecrypt_UnknownCryptoTypeFallsBackToLegacy(t *testing.T) {
	uuid := "u-legacy"
	password := "p-legacy"
	plain := DecryptedData{
		CookieData:       map[string][]Cookie{"example.com": {{Name: "sid", Value: "x"}}},
		LocalStorageData: map[string]map[string]string{},
		UpdateTime:       "2026-02-01T00:00:00Z",
	}
	plainBytes, err := json.Marshal(plain)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	encrypted := encryptLegacy(t, uuid, password, plainBytes, []byte("12345678"))

	got, err := Decrypt(uuid, encrypted, password, CryptoType("unknown-type"))
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if len(got.CookieData["example.com"]) != 1 {
		t.Fatalf("unexpected decrypt result: %+v", got.CookieData)
	}
}

func TestDecryptFixed_InvalidAESKeyLength(t *testing.T) {
	ciphertext := make([]byte, 16)
	encrypted := base64.StdEncoding.EncodeToString(ciphertext)
	if _, err := decryptFixed([]byte("short"), encrypted); err == nil || !strings.Contains(err.Error(), "aes new cipher") {
		t.Fatalf("expected aes new cipher error, got=%v", err)
	}
}

func TestPKCS7Unpad_MoreErrorBranches(t *testing.T) {
	if _, err := pkcs7Unpad([]byte{1, 2, 3, 4}, 0); err == nil || !strings.Contains(err.Error(), "invalid block size") {
		t.Fatalf("expected invalid block size error, got=%v", err)
	}
	if _, err := pkcs7Unpad([]byte{1, 2, 3, 0}, 4); err == nil || !strings.Contains(err.Error(), "invalid padding") {
		t.Fatalf("expected invalid padding error for pad=0, got=%v", err)
	}
	if _, err := pkcs7Unpad([]byte{1, 2, 3, 5}, 4); err == nil || !strings.Contains(err.Error(), "invalid padding") {
		t.Fatalf("expected invalid padding error for pad>block, got=%v", err)
	}
	if _, err := pkcs7Unpad([]byte{1, 2, 3, 2}, 4); err == nil || !strings.Contains(err.Error(), "invalid padding") {
		t.Fatalf("expected invalid padding error for tail mismatch, got=%v", err)
	}
}

func TestClient_GetEncrypted_RequestBuildError(t *testing.T) {
	c := &Client{baseURL: &url.URL{Scheme: "http", Host: "bad host"}, httpClient: http.DefaultClient}
	if _, err := c.GetEncrypted(context.Background(), "u1", ""); err == nil || !strings.Contains(err.Error(), "build request") {
		t.Fatalf("expected build request error, got=%v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_GetEncrypted_RequestError(t *testing.T) {
	c, err := NewClient("https://example.com/api")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.WithHTTPClient(&http.Client{Transport: roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("dial boom")
	})})

	if _, err := c.GetEncrypted(context.Background(), "u1", ""); err == nil || !strings.Contains(err.Error(), "request") {
		t.Fatalf("expected request error, got=%v", err)
	}
}

func TestClient_GetDecrypted_ErrorAndServerCryptoFallback(t *testing.T) {
	t.Run("get encrypted returns error", func(t *testing.T) {
		c, err := NewClient("https://example.com/api")
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		if _, err := c.GetDecrypted(context.Background(), "", "pwd", ""); err == nil {
			t.Fatalf("expected error when uuid is empty")
		}
	})

	t.Run("use server crypto type when override is empty", func(t *testing.T) {
		uuid := "u2"
		password := "p2"
		plain := DecryptedData{
			CookieData:       map[string][]Cookie{"example.com": {{Name: "a", Value: "1"}}},
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
			_ = json.NewEncoder(w).Encode(GetResponse{Encrypted: encrypted, CryptoType: CryptoTypeAES128CBCFixed})
		}))
		defer srv.Close()

		c, err := NewClient(srv.URL)
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		got, err := c.GetDecrypted(context.Background(), uuid, password, "")
		if err != nil {
			t.Fatalf("GetDecrypted: %v", err)
		}
		if len(got.CookieData["example.com"]) != 1 {
			t.Fatalf("unexpected decrypt result: %+v", got.CookieData)
		}
	})
}

func TestClient_GetCookieHeader_DecryptError(t *testing.T) {
	c, err := NewClient("https://example.com/api")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.GetCookieHeader(context.Background(), "", "pwd", "example.com", ""); err == nil {
		t.Fatalf("expected decrypt path error for empty uuid")
	}
}

func TestNewClient_ParseError(t *testing.T) {
	if _, err := NewClient("https://%zz"); err == nil {
		t.Fatalf("want parse error for malformed URL escape")
	}
}

func TestDecryptFixed_UnpadError(t *testing.T) {
	uuid := "u-fixed-unpad"
	password := "p-fixed-unpad"
	key := derivePassphrase(uuid, password)
	block, err := aesNewCipher(key)
	if err != nil {
		t.Fatalf("aesNewCipher: %v", err)
	}

	// 构造 16 字节非 PKCS7 填充数据（末尾 0），用于触发 unpad 错误分支。
	plain := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0}
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, make([]byte, 16)).CryptBlocks(ciphertext, plain)
	encrypted := base64.StdEncoding.EncodeToString(ciphertext)

	if _, err := decryptFixed(key, encrypted); err == nil || !strings.Contains(err.Error(), "unpad (fixed)") {
		t.Fatalf("expected fixed unpad error, got=%v", err)
	}
}

func TestDecryptLegacy_MoreErrorBranches(t *testing.T) {
	passphrase := derivePassphrase("u-legacy-err", "p-legacy-err")

	t.Run("base64 decode error", func(t *testing.T) {
		if _, err := decryptLegacy(passphrase, "%%%not-base64%%%"); err == nil || !strings.Contains(err.Error(), "base64 decode (legacy)") {
			t.Fatalf("expected legacy base64 decode error, got=%v", err)
		}
	})

	t.Run("invalid ciphertext length", func(t *testing.T) {
		raw := append([]byte(opensslSaltPrefix), []byte("12345678")...)
		raw = append(raw, 1) // 仅 1 字节密文，必然不是 16 的倍数
		encrypted := base64.StdEncoding.EncodeToString(raw)
		if _, err := decryptLegacy(passphrase, encrypted); err == nil || !strings.Contains(err.Error(), "invalid ciphertext length (legacy)") {
			t.Fatalf("expected legacy ciphertext length error, got=%v", err)
		}
	})

	t.Run("legacy unpad error", func(t *testing.T) {
		salt := []byte("12345678")
		key, iv := evpBytesToKeyMD5(passphrase, salt, 32, 16)
		block, err := aesNewCipher(key)
		if err != nil {
			t.Fatalf("aesNewCipher: %v", err)
		}
		// 末尾 0 不是合法 padding。
		plain := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 0}
		ciphertext := make([]byte, len(plain))
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plain)

		raw := make([]byte, 0, 16+len(ciphertext))
		raw = append(raw, []byte(opensslSaltPrefix)...)
		raw = append(raw, salt...)
		raw = append(raw, ciphertext...)

		encrypted := base64.StdEncoding.EncodeToString(raw)
		if _, err := decryptLegacy(passphrase, encrypted); err == nil || !strings.Contains(err.Error(), "unpad (legacy)") {
			t.Fatalf("expected legacy unpad error, got=%v", err)
		}
	})

	t.Run("aes new cipher error via hook", func(t *testing.T) {
		old := aesNewCipher
		aesNewCipher = func([]byte) (cipher.Block, error) {
			return nil, errors.New("hooked legacy cipher error")
		}
		t.Cleanup(func() { aesNewCipher = old })

		salt := []byte("12345678")
		// 构造最小合法格式，确保流程走到 aesNewCipher。
		raw := make([]byte, 0, 32)
		raw = append(raw, []byte(opensslSaltPrefix)...)
		raw = append(raw, salt...)
		raw = append(raw, make([]byte, 16)...)
		encrypted := base64.StdEncoding.EncodeToString(raw)

		if _, err := decryptLegacy(passphrase, encrypted); err == nil || !strings.Contains(err.Error(), "aes new cipher (legacy)") {
			t.Fatalf("expected legacy aes new cipher error, got=%v", err)
		}
	})
}
