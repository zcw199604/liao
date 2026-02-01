package cookiecloud

// CryptoType controls how CookieCloud data is encrypted/decrypted.
// CookieCloud currently uses "legacy" or "aes-128-cbc-fixed".
type CryptoType string

const (
	CryptoTypeLegacy         CryptoType = "legacy"
	CryptoTypeAES128CBCFixed            = "aes-128-cbc-fixed"
)

// GetResponse matches CookieCloud's /get/:uuid response when returning encrypted data.
type GetResponse struct {
	Encrypted  string     `json:"encrypted"`
	CryptoType CryptoType `json:"crypto_type"`
}

// Cookie mirrors the browser cookie objects that CookieCloud stores.
// We intentionally only model fields we need (name/value) plus a few common extras.
type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HttpOnly bool   `json:"httpOnly,omitempty"`
	SameSite string `json:"sameSite,omitempty"`
}

// DecryptedData is the plaintext JSON payload stored by CookieCloud.
type DecryptedData struct {
	CookieData       map[string][]Cookie          `json:"cookie_data"`
	LocalStorageData map[string]map[string]string `json:"local_storage_data"`
	UpdateTime       string                       `json:"update_time"`
}
