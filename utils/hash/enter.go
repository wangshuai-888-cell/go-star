package hash

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// NewRefreshToken 生成高熵 refresh token（明文只返回给前端一次）
func NewRefreshToken() (string, error) {
	b := make([]byte, 32) // 256 bit
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SHA256Hex 对明文做 sha256，存库用（64 位 hex，对应 gorm size:64）
func SHA256Hex(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
