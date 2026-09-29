package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// SessionCookie is the only value stored in the browser.
const SessionCookie = "sitewise_session"

// NewToken returns a raw invite token and the SHA-256 stored at rest.
func NewToken() (string, []byte, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}
	raw := hex.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

// HashToken is the value persisted for an invite. The raw token is not stored.
func HashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
