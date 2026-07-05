package filestore

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"
)

func AssetURL(baseURL string, key string) (string, error) {
	cleanKey, err := CleanKey(key)
	if err != nil {
		return "", err
	}
	return joinURL(baseURL, "assets", escapeKey(cleanKey)), nil
}

func SignedTestfileURL(baseURL string, sha256Hex string, signingKey []byte, expires time.Time) string {
	key := fmt.Sprintf("%s.zst", sha256Hex)
	expiresUnix := expires.Unix()
	sig := SignTestfile(sha256Hex, expiresUnix, signingKey)
	u := joinURL(baseURL, "testfiles", key)
	return fmt.Sprintf("%s?expires=%d&sig=%s", u, expiresUnix, sig)
}

func SignTestfile(sha256Hex string, expiresUnix int64, signingKey []byte) string {
	mac := hmac.New(sha256.New, signingKey)
	_, _ = fmt.Fprintf(mac, "%s:%d", sha256Hex, expiresUnix)
	return hex.EncodeToString(mac.Sum(nil))
}

func ValidTestfileSignature(sha256Hex string, expiresUnix int64, signature string, signingKey []byte, now time.Time) bool {
	if now.Unix() > expiresUnix {
		return false
	}
	expected := SignTestfile(sha256Hex, expiresUnix, signingKey)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func NormalizeBaseURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/")
}

func joinURL(baseURL string, parts ...string) string {
	u := NormalizeBaseURL(baseURL)
	for _, part := range parts {
		u += "/" + strings.Trim(part, "/")
	}
	return u
}

func escapeKey(key string) string {
	parts := strings.Split(key, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return path.Join(parts...)
}
