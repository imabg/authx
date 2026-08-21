package mail

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/imabg/authx/internal/secret"
)

func TestDecodeSMTPPassword(t *testing.T) {
	got, err := DecodeSMTPPassword(base64.StdEncoding.EncodeToString([]byte("s3cret")))
	if err != nil {
		t.Fatalf("DecodeSMTPPassword: %v", err)
	}
	if got != "s3cret" {
		t.Fatalf("got %q", got)
	}
	if _, err := DecodeSMTPPassword("not-base64!"); err == nil || !strings.Contains(err.Error(), "mail.smtp.password must be base64-encoded") {
		t.Fatalf("error = %v", err)
	}
	got, err = DecodeSMTPPassword("")
	if err != nil || got != "" {
		t.Fatalf("empty = %q %v", got, err)
	}
}

func TestEncryptDecryptSecrets(t *testing.T) {
	box, err := secret.NewBox("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	cfg := Config{SMTP: SMTPConfig{Username: "acme", Password: "s3cret"}}
	if err := cfg.EncryptSecrets(box); err != nil {
		t.Fatalf("EncryptSecrets: %v", err)
	}
	if cfg.SMTP.Username == "acme" || cfg.SMTP.Password == "s3cret" {
		t.Fatal("expected credentials to be encrypted")
	}
	if !secret.IsEncrypted(cfg.SMTP.Username) || !secret.IsEncrypted(cfg.SMTP.Password) {
		t.Fatalf("encrypted values = %+v", cfg.SMTP)
	}
	if err := cfg.DecryptSecrets(box); err != nil {
		t.Fatalf("DecryptSecrets: %v", err)
	}
	if cfg.SMTP.Username != "acme" || cfg.SMTP.Password != "s3cret" {
		t.Fatalf("decrypted = %+v", cfg.SMTP)
	}
}
