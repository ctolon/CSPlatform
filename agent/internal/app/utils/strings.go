package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

func HasAnyPrefix(s string, prefs []string) bool {
	for _, p := range prefs {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func HasAnySuffix(s string, sufs []string) bool {
	for _, x := range sufs {
		if strings.HasSuffix(s, x) {
			return true
		}
	}
	return false
}

func ParseSearchAttr(attrs string) []string {
	if attrs == "" {
		return nil
	}
	if strings.Contains(attrs, ",") {
		return strings.Split(attrs, ",")
	}
	return []string{attrs}
}

func ParseToList(s string) []string {
	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	return []string{strings.TrimSpace(s)}
}

// NewEncKey32FromSecret returns a 32-byte AES key.
// Accepts Base64 (std/raw) or plain string; always derives 32 bytes via SHA-256.
func NewEncKey32FromSecret(secret string) []byte {
	if secret == "" {
		panic("secret empty")
	}
	if b, err := base64.StdEncoding.DecodeString(secret); err == nil {
		sum := sha256.Sum256(b)
		return sum[:]
	}
	if b, err := base64.RawStdEncoding.DecodeString(secret); err == nil {
		sum := sha256.Sum256(b)
		return sum[:]
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
