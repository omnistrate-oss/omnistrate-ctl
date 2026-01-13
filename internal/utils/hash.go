package utils

import (
	"crypto/sha256"
	"fmt"
)

// HashSha256 returns the SHA256 hash of the given string.
// This is used to hash file contents or other strings for integrity checks.
// Not used for password hashing.
func HashSha256(source string) (hash string) {
	h := sha256.Sum256([]byte(source))
	hash = fmt.Sprintf("%x", h[:])
	return
}
