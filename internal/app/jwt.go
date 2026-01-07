package app

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService 提供与 Spring 侧兼容的 JWT 生成与校验（HS256，sub=user）。
type JWTService struct {
	secret []byte
	expire time.Duration
}

func NewJWTService(secret string, expireHours int) *JWTService {
	if expireHours <= 0 {
		expireHours = 24
	}
	return &JWTService{
		secret: []byte(secret),
		expire: time.Duration(expireHours) * time.Hour,
	}
}

func (s *JWTService) GenerateToken() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   "user",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.expire)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("签发 Token 失败: %w", err)
	}
	return signed, nil
}

func (s *JWTService) ValidateToken(tokenString string) bool {
	if tokenString == "" {
		return false
	}

	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("不支持的签名算法: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	return err == nil
}

