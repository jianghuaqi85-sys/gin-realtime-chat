package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrTokenExpired = errors.New("token has expired")
	ErrTokenInvalid = errors.New("token is invalid")
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret []byte
	issuer string
}

func NewManager(secret string, issuer string) (*Manager, error) {
	if secret == "" {
		return nil, fmt.Errorf("JWT secret must not be empty")
	}
	return &Manager{
		secret: []byte(secret),
		issuer: issuer,
	}, nil
}

func (m *Manager) GenerateToken(userID, username, role string, expireHours int) (string, error) {
	now := time.Now()
	expireTime := now.Add(time.Duration(expireHours) * time.Hour)

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(now),
			// NotBefore 往前防范 5 秒的时钟偏移
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			// 注入 UUID 作为 JTI 防重放与支持 Token 撤销
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// 兼容性的顶级全局函数封装

func GenerateToken(secret string, userID, username, role string, expireHours int) (string, error) {
	mgr, err := NewManager(secret, "gin-high-performance")
	if err != nil {
		return "", err
	}
	return mgr.GenerateToken(userID, username, role, expireHours)
}

func ParseToken(tokenStr string, secret string) (*Claims, error) {
	mgr, err := NewManager(secret, "gin-high-performance")
	if err != nil {
		return nil, err
	}
	return mgr.ParseToken(tokenStr)
}

func ValidateToken(tokenStr string, secret string) (*Claims, error) {
	return ParseToken(tokenStr, secret)
}
