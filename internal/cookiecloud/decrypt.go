package cookiecloud

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

const opensslSaltPrefix = "Salted__"

// Decrypt decrypts CookieCloud data into its plaintext JSON structure.
//
// If cryptoType is empty/unknown, it falls back to legacy for compatibility.
func Decrypt(uuid, encrypted, password string, cryptoType CryptoType) (*DecryptedData, error) {
	passphrase := derivePassphrase(uuid, password)

	var plain []byte
	var err error
	if cryptoType == CryptoTypeAES128CBCFixed {
		plain, err = decryptFixed(passphrase, encrypted)
	} else {
		plain, err = decryptLegacy(passphrase, encrypted)
	}
	if err != nil {
		return nil, err
	}

	var out DecryptedData
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, fmt.Errorf("cookiecloud: json unmarshal: %w", err)
	}
	return &out, nil
}

func derivePassphrase(uuid, password string) []byte {
	sum := md5.Sum([]byte(uuid + "-" + password))
	hexStr := hex.EncodeToString(sum[:])
	// CookieCloud uses the first 16 chars of the MD5 hex string as the passphrase/key material.
	return []byte(hexStr[:16])
}

func decryptFixed(key []byte, encrypted string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("cookiecloud: base64 decode (fixed): %w", err)
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("cookiecloud: invalid ciphertext length (fixed): %d", len(ciphertext))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cookiecloud: aes new cipher (fixed): %w", err)
	}

	iv := make([]byte, aes.BlockSize) // 16 bytes of 0x00
	mode := cipher.NewCBCDecrypter(block, iv)

	plain := make([]byte, len(ciphertext))
	mode.CryptBlocks(plain, ciphertext)

	plain, err = pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return nil, fmt.Errorf("cookiecloud: unpad (fixed): %w", err)
	}
	return plain, nil
}

func decryptLegacy(passphrase []byte, encrypted string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("cookiecloud: base64 decode (legacy): %w", err)
	}
	if len(raw) < 16 {
		return nil, fmt.Errorf("cookiecloud: legacy payload too short: %d", len(raw))
	}
	if string(raw[:8]) != opensslSaltPrefix {
		return nil, fmt.Errorf("cookiecloud: legacy payload missing OpenSSL salt prefix")
	}

	salt := raw[8:16]
	ciphertext := raw[16:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("cookiecloud: invalid ciphertext length (legacy): %d", len(ciphertext))
	}

	key, iv := evpBytesToKeyMD5(passphrase, salt, 32, aes.BlockSize)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cookiecloud: aes new cipher (legacy): %w", err)
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plain := make([]byte, len(ciphertext))
	mode.CryptBlocks(plain, ciphertext)

	plain, err = pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return nil, fmt.Errorf("cookiecloud: unpad (legacy): %w", err)
	}
	return plain, nil
}

func evpBytesToKeyMD5(passphrase, salt []byte, keyLen, ivLen int) ([]byte, []byte) {
	totalLen := keyLen + ivLen
	var out []byte

	var prev []byte
	for len(out) < totalLen {
		h := md5.New()
		h.Write(prev)
		h.Write(passphrase)
		h.Write(salt)
		prev = h.Sum(nil)
		out = append(out, prev...)
	}

	key := out[:keyLen]
	iv := out[keyLen:totalLen]
	return key, iv
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}
	if blockSize <= 0 {
		return nil, errors.New("invalid block size")
	}
	if len(data)%blockSize != 0 {
		return nil, errors.New("data is not aligned to block size")
	}

	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, errors.New("invalid padding")
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil, errors.New("invalid padding")
		}
	}
	return data[:len(data)-padLen], nil
}
