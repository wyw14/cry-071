package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type TokenCodec struct{ pepper []byte }

func NewTokenCodec(pepper string) (*TokenCodec, error) {
	pepper = strings.TrimSpace(pepper)
	if len(pepper) < 12 {
		return nil, fmt.Errorf("query token pepper must have at least 12 characters")
	}
	return &TokenCodec{pepper: []byte(pepper)}, nil
}

func (c *TokenCodec) Issue(feedbackID string, expiresAt time.Time) (string, string, error) {
	if strings.TrimSpace(feedbackID) == "" || expiresAt.IsZero() {
		return "", "", fmt.Errorf("token subject and expiry are required")
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", "", fmt.Errorf("generate query token: %w", err)
	}
	plain := base64.RawURLEncoding.EncodeToString(random)
	return plain, c.Digest(plain), nil
}

func (c *TokenCodec) Digest(plain string) string {
	mac := hmac.New(sha256.New, c.pepper)
	_, _ = mac.Write([]byte(strings.TrimSpace(plain)))
	return hex.EncodeToString(mac.Sum(nil))
}
