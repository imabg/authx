package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const prefix = "enc:v1:"

// Box encrypts small secrets at rest (AES-256-GCM).
type Box interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type aesBox struct {
	gcm cipher.AEAD
}

type disabledBox struct{}

func IsEncrypted(v string) bool {
	return strings.HasPrefix(v, prefix)
}

// NewBox builds an AES-256-GCM box from a 32-byte key.
// The key may be 64 hex characters or standard base64. An empty key returns a
// disabled box that only allows empty secrets.
func NewBox(key string) (Box, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Disabled(), nil
	}
	raw, err := parseKey(key)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, fmt.Errorf("encryption.key: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encryption.key: %w", err)
	}
	return &aesBox{gcm: gcm}, nil
}

func Disabled() Box {
	return disabledBox{}
}

func parseKey(s string) ([]byte, error) {
	if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) == 32 {
		return b, nil
	}
	return nil, fmt.Errorf("encryption.key must be 32 bytes as hex or base64")
}

func (b *aesBox) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if IsEncrypted(plaintext) {
		return plaintext, nil
	}
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := b.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

func (b *aesBox) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !IsEncrypted(ciphertext) {
		// Legacy plaintext rows (pre-encryption) remain usable until next save.
		return ciphertext, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, prefix))
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	nonceSize := b.gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("decrypt secret: ciphertext too short")
	}
	plain, err := b.gcm.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plain), nil
}

func (disabledBox) Encrypt(plaintext string) (string, error) {
	if strings.TrimSpace(plaintext) == "" {
		return "", nil
	}
	if IsEncrypted(plaintext) {
		return plaintext, nil
	}
	return "", fmt.Errorf("encryption.key is not configured")
}

func (disabledBox) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if IsEncrypted(ciphertext) {
		return "", fmt.Errorf("encryption.key is not configured")
	}
	return ciphertext, nil
}
