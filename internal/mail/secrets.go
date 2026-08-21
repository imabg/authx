package mail

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/imabg/authx/internal/secret"
)

func DecodeSMTPPassword(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(v)
		if err != nil {
			return "", fmt.Errorf("mail.smtp.password must be base64-encoded")
		}
	}
	return string(decoded), nil
}

// DecodeIncomingSMTP decodes a client-supplied base64 SMTP password in place.
// Encrypted-at-rest values are left unchanged so DB loads are not re-decoded.
func DecodeIncomingSMTP(cfg *Config) error {
	if cfg == nil || SecretUnchanged(cfg.SMTP.Password) {
		return nil
	}
	if secret.IsEncrypted(cfg.SMTP.Password) {
		return nil
	}
	decoded, err := DecodeSMTPPassword(cfg.SMTP.Password)
	if err != nil {
		return err
	}
	cfg.SMTP.Password = decoded
	return nil
}

func (c *Config) EncryptSecrets(box secret.Box) error {
	if c == nil {
		return nil
	}
	user, err := encryptField(box, c.SMTP.Username)
	if err != nil {
		return err
	}
	pass, err := encryptField(box, c.SMTP.Password)
	if err != nil {
		return err
	}
	c.SMTP.Username = user
	c.SMTP.Password = pass
	return nil
}

func (c *Config) DecryptSecrets(box secret.Box) error {
	if c == nil {
		return nil
	}
	user, err := decryptField(box, c.SMTP.Username)
	if err != nil {
		return err
	}
	pass, err := decryptField(box, c.SMTP.Password)
	if err != nil {
		return err
	}
	c.SMTP.Username = user
	c.SMTP.Password = pass
	return nil
}

func encryptField(box secret.Box, v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}
	if secret.IsEncrypted(v) {
		return v, nil
	}
	if box == nil {
		box = secret.Disabled()
	}
	return box.Encrypt(v)
}

func decryptField(box secret.Box, v string) (string, error) {
	if v == "" {
		return "", nil
	}
	if box == nil {
		box = secret.Disabled()
	}
	return box.Decrypt(v)
}
