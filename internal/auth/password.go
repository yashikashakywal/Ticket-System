package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Password hashing is implemented with PBKDF2-HMAC-SHA256 using only the Go
// standard library (crypto/hmac, crypto/sha256). This avoids pulling in
// golang.org/x/crypto/bcrypt as an external module dependency, while still
// storing salted, iterated, non-reversible hashes instead of plain text.

const (
	pbkdf2Iterations = 100_000
	pbkdf2KeyLen     = 32
	saltLen          = 16
)

// HashPassword returns a self-describing hash string of the form:
// pbkdf2$<iterations>$<base64-salt>$<base64-hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	hash := pbkdf2(password, salt, pbkdf2Iterations, pbkdf2KeyLen)
	encoded := fmt.Sprintf("pbkdf2$%d$%s$%s",
		pbkdf2Iterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword checks a plaintext password against a stored hash produced
// by HashPassword. It returns true only on an exact, constant-time match.
func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got := pbkdf2(password, salt, iterations, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// pbkdf2 implements RFC 8018 PBKDF2 using HMAC-SHA256 as the PRF.
func pbkdf2(password string, salt []byte, iterations, keyLen int) []byte {
	prf := func() hashFunc { return hmac.New(sha256.New, []byte(password)) }
	hashLen := sha256.Size
	numBlocks := (keyLen + hashLen - 1) / hashLen

	derived := make([]byte, 0, numBlocks*hashLen)
	for block := 1; block <= numBlocks; block++ {
		h := prf()
		h.Write(salt)
		h.Write(intToBytes(uint32(block)))
		u := h.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)

		for i := 1; i < iterations; i++ {
			h := prf()
			h.Write(u)
			u = h.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		derived = append(derived, t...)
	}
	return derived[:keyLen]
}

// hashFunc is the minimal interface pbkdf2 needs from hmac.New's return value.
type hashFunc interface {
	Write(p []byte) (n int, err error)
	Sum(b []byte) []byte
}

func intToBytes(i uint32) []byte {
	return []byte{byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)}
}
