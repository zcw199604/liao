package cookiecloud

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestDecrypt_AES128CBCFixed_RoundTrip(t *testing.T) {
	uuid := "test-uuid"
	password := "test-password"

	want := DecryptedData{
		CookieData: map[string][]Cookie{
			".example.com": {
				{Name: "a", Value: "1", Domain: ".example.com", Path: "/", Secure: true, HttpOnly: true},
				{Name: "b", Value: "2", Domain: ".example.com", Path: "/"},
			},
		},
		LocalStorageData: map[string]map[string]string{
			"example.com": {"k": "v"},
		},
		UpdateTime: "2026-02-01T00:00:00Z",
	}

	plain, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal plaintext: %v", err)
	}
	encrypted := encryptFixed(t, uuid, password, plain)

	got, err := Decrypt(uuid, encrypted, password, CryptoTypeAES128CBCFixed)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !reflect.DeepEqual(want, *got) {
		t.Fatalf("decrypted mismatch\nwant=%#v\ngot=%#v", want, *got)
	}
}

func TestDecrypt_Legacy_RoundTrip(t *testing.T) {
	uuid := "test-uuid"
	password := "test-password"

	want := DecryptedData{
		CookieData: map[string][]Cookie{
			"example.com": {
				{Name: "sid", Value: "abc"},
			},
		},
		LocalStorageData: map[string]map[string]string{},
		UpdateTime:       "2026-02-01T00:00:00Z",
	}

	plain, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal plaintext: %v", err)
	}
	encrypted := encryptLegacy(t, uuid, password, plain, []byte("12345678"))

	got, err := Decrypt(uuid, encrypted, password, CryptoTypeLegacy)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !reflect.DeepEqual(want, *got) {
		t.Fatalf("decrypted mismatch\nwant=%#v\ngot=%#v", want, *got)
	}
}

func TestClient_GetCookieHeader(t *testing.T) {
	uuid := "u1"
	password := "p1"

	plain := DecryptedData{
		CookieData: map[string][]Cookie{
			".example.com": {
				{Name: "a", Value: "1"},
				{Name: "b", Value: "2"},
			},
		},
		LocalStorageData: map[string]map[string]string{},
		UpdateTime:       "2026-02-01T00:00:00Z",
	}
	plainBytes, err := json.Marshal(plain)
	if err != nil {
		t.Fatalf("marshal plaintext: %v", err)
	}
	enc := encryptFixed(t, uuid, password, plainBytes)

	type reqInfo struct {
		Path       string
		CryptoType string
	}
	reqCh := make(chan reqInfo, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/get/"+uuid, func(w http.ResponseWriter, r *http.Request) {
		reqCh <- reqInfo{
			Path:       r.URL.Path,
			CryptoType: r.URL.Query().Get("crypto_type"),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GetResponse{
			Encrypted:  enc,
			CryptoType: CryptoTypeAES128CBCFixed,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL + "/api")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	gotHeader, err := c.GetCookieHeader(context.Background(), uuid, password, "example.com", CryptoTypeAES128CBCFixed)
	if err != nil {
		t.Fatalf("GetCookieHeader: %v", err)
	}
	if gotHeader != "a=1; b=2" {
		t.Fatalf("cookie header mismatch: %q", gotHeader)
	}

	ri := <-reqCh
	if ri.Path != "/api/get/"+uuid {
		t.Fatalf("unexpected request path: %q", ri.Path)
	}
	if ri.CryptoType != string(CryptoTypeAES128CBCFixed) {
		t.Fatalf("unexpected crypto_type query: %q", ri.CryptoType)
	}
}

func encryptFixed(t *testing.T, uuid, password string, plain []byte) string {
	t.Helper()

	key := derivePassphrase(uuid, password)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}

	iv := make([]byte, aes.BlockSize) // 16 bytes of 0x00
	mode := cipher.NewCBCEncrypter(block, iv)

	padded := pkcs7Pad(plain, aes.BlockSize)
	out := make([]byte, len(padded))
	mode.CryptBlocks(out, padded)

	return base64.StdEncoding.EncodeToString(out)
}

func encryptLegacy(t *testing.T, uuid, password string, plain []byte, salt []byte) string {
	t.Helper()
	if len(salt) != 8 {
		t.Fatalf("salt must be 8 bytes, got %d", len(salt))
	}

	passphrase := derivePassphrase(uuid, password)
	key, iv := evpBytesToKeyMD5(passphrase, salt, 32, aes.BlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	padded := pkcs7Pad(plain, aes.BlockSize)
	ct := make([]byte, len(padded))
	mode.CryptBlocks(ct, padded)

	raw := make([]byte, 0, 16+len(ct))
	raw = append(raw, []byte(opensslSaltPrefix)...)
	raw = append(raw, salt...)
	raw = append(raw, ct...)

	return base64.StdEncoding.EncodeToString(raw)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	out := make([]byte, len(data)+padLen)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padLen)
	}
	return out
}
