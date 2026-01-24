package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewJWTService_DefaultExpireHours(t *testing.T) {
	svc := NewJWTService("secret", 0)
	if svc.expire != 24*time.Hour {
		t.Fatalf("expire=%v, want %v", svc.expire, 24*time.Hour)
	}
}

func TestJWTService_GenerateToken_EmptySecret(t *testing.T) {
	svc := NewJWTService("", 24)
	if _, err := svc.GenerateToken(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestJWTService_GenerateToken_SignedStringFailed(t *testing.T) {
	old := jwtSignedStringFn
	t.Cleanup(func() { jwtSignedStringFn = old })
	jwtSignedStringFn = func(token *jwt.Token, key any) (string, error) {
		return "", errors.New("sign fail")
	}

	svc := NewJWTService("secret", 24)
	if _, err := svc.GenerateToken(); err == nil || !strings.Contains(err.Error(), "签发 Token 失败") {
		t.Fatalf("err=%v", err)
	}
}

func TestJWTService_ValidateToken_EmptyTokenString(t *testing.T) {
	svc := NewJWTService("secret", 24)
	if svc.ValidateToken("") {
		t.Fatalf("expected invalid token")
	}
}

func TestJWTService_ValidateToken_EmptySecret(t *testing.T) {
	svc := NewJWTService("", 24)
	if svc.ValidateToken("token") {
		t.Fatalf("expected invalid token")
	}
}

func TestJWTService_ValidateToken_Expired(t *testing.T) {
	svc := NewJWTService("secret", 24)

	claims := jwt.RegisteredClaims{
		Subject:   "user",
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if svc.ValidateToken(signed) {
		t.Fatalf("expected invalid token")
	}
}

func TestJWTService_ValidateToken_AlgorithmMismatch(t *testing.T) {
	svc := NewJWTService("secret", 24)

	claims := jwt.RegisteredClaims{
		Subject:   "user",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if svc.ValidateToken(signed) {
		t.Fatalf("expected invalid token")
	}
}

func TestJWTService_GenerateAndValidate_Success(t *testing.T) {
	svc := NewJWTService("secret", 24)
	signed, err := svc.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if !svc.ValidateToken(signed) {
		t.Fatalf("expected token valid")
	}
}
