package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// This is a minimal, dependency-free HS256 JWT implementation (header +
// claims + HMAC-SHA256 signature), built entirely on the standard library.
// It supports exactly what this service needs: issuing and verifying signed
// tokens carrying a user id, email, and expiry.

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims carried inside the JWT payload.
type Claims struct {
	UserID    string `json:"sub"`
	Email     string `json:"email"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func b64Encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func b64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// GenerateToken creates a signed JWT for the given user, valid for ttl.
func GenerateToken(secret, userID, email string, ttl time.Duration) (string, error) {
	now := time.Now()
	h := header{Alg: "HS256", Typ: "JWT"}
	c := Claims{
		UserID:    userID,
		Email:     email,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}

	headerJSON, err := json.Marshal(h)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(c)
	if err != nil {
		return "", err
	}

	unsigned := b64Encode(headerJSON) + "." + b64Encode(claimsJSON)
	sig := sign(unsigned, secret)
	return unsigned + "." + b64Encode(sig), nil
}

// ParseToken verifies the signature and expiry of a token and returns its claims.
func ParseToken(secret, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	unsigned := parts[0] + "." + parts[1]

	sig, err := b64Decode(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	expectedSig := sign(unsigned, secret)
	if subtle.ConstantTimeCompare(sig, expectedSig) != 1 {
		return nil, ErrInvalidToken
	}

	claimsJSON, err := b64Decode(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(claimsJSON, &c); err != nil {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() > c.ExpiresAt {
		return nil, ErrExpiredToken
	}
	return &c, nil
}

func sign(unsigned, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return mac.Sum(nil)
}
