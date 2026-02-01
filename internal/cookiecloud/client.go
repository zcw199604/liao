package cookiecloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client talks to a CookieCloud server and decrypts the returned payload locally.
//
// It intentionally does NOT support server-side decrypt (sending password to the server).
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("cookiecloud: empty baseURL")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("cookiecloud: parse baseURL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("cookiecloud: baseURL must include scheme and host: %q", baseURL)
	}
	return &Client{
		baseURL:    u,
		httpClient: http.DefaultClient,
	}, nil
}

func (c *Client) WithHTTPClient(hc *http.Client) *Client {
	if hc != nil {
		c.httpClient = hc
	}
	return c
}

// GetEncrypted fetches /get/:uuid and returns the server's encrypted payload.
//
// If cryptoType is non-empty, it is passed as ?crypto_type=... for compatibility.
func (c *Client) GetEncrypted(ctx context.Context, uuid string, cryptoType CryptoType) (*GetResponse, error) {
	if strings.TrimSpace(uuid) == "" {
		return nil, fmt.Errorf("cookiecloud: empty uuid")
	}

	u := c.baseURL.JoinPath("get", uuid)
	if cryptoType != "" {
		q := u.Query()
		q.Set("crypto_type", string(cryptoType))
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("cookiecloud: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cookiecloud: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("cookiecloud: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out GetResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("cookiecloud: decode response: %w", err)
	}
	if strings.TrimSpace(out.Encrypted) == "" {
		return nil, fmt.Errorf("cookiecloud: missing encrypted in response")
	}
	return &out, nil
}

// GetDecrypted fetches encrypted data and decrypts it locally.
//
// cryptoTypeOverride:
// - If non-empty, it is used for decryption (and also passed as query ?crypto_type=...).
// - Otherwise, it uses the server-provided crypto_type (or falls back to legacy).
func (c *Client) GetDecrypted(ctx context.Context, uuid, password string, cryptoTypeOverride CryptoType) (*DecryptedData, error) {
	enc, err := c.GetEncrypted(ctx, uuid, cryptoTypeOverride)
	if err != nil {
		return nil, err
	}

	useCrypto := cryptoTypeOverride
	if useCrypto == "" {
		useCrypto = enc.CryptoType
	}

	return Decrypt(uuid, enc.Encrypted, password, useCrypto)
}

// GetCookieHeader fetches and decrypts CookieCloud data and returns a "Cookie" header value
// for a specific domain (e.g. "example.com").
func (c *Client) GetCookieHeader(ctx context.Context, uuid, password, domain string, cryptoTypeOverride CryptoType) (string, error) {
	data, err := c.GetDecrypted(ctx, uuid, password, cryptoTypeOverride)
	if err != nil {
		return "", err
	}
	cookies := FindCookiesForDomain(data.CookieData, domain)
	if len(cookies) == 0 {
		return "", fmt.Errorf("cookiecloud: no cookies for domain %q", domain)
	}
	return BuildCookieHeader(cookies), nil
}
