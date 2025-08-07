package utils

import (
	"crypto/sha256"
	"fmt"
)

func HashSha256(password string) (hash string) {
	h := sha256.Sum256([]byte(password))
	hash = fmt.Sprintf("%x", h[:])
	return
}
