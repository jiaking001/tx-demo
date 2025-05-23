package pkg

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashPassword 密码哈希
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}
