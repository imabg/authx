package secret

import (
	"strings"
	"testing"
)

func TestBoxRoundTrip(t *testing.T) {
	box, err := NewBox("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	ct, err := box.Encrypt("smtp-password")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !IsEncrypted(ct) || strings.Contains(ct, "smtp-password") {
		t.Fatalf("ciphertext = %q", ct)
	}
	got, err := box.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "smtp-password" {
		t.Fatalf("plaintext = %q", got)
	}
	again, err := box.Encrypt(ct)
	if err != nil {
		t.Fatalf("Encrypt ciphertext: %v", err)
	}
	if again != ct {
		t.Fatal("Encrypt should be idempotent for already-encrypted values")
	}
}

func TestBoxDecryptsLegacyPlaintext(t *testing.T) {
	box, err := NewBox("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	got, err := box.Decrypt("legacy-secret")
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "legacy-secret" {
		t.Fatalf("got %q", got)
	}
}

func TestNewBoxRejectsBadKey(t *testing.T) {
	if _, err := NewBox("too-short"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDisabledBox(t *testing.T) {
	box, err := NewBox("")
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	if _, err := box.Encrypt("secret"); err == nil || !strings.Contains(err.Error(), "encryption.key is not configured") {
		t.Fatalf("error = %v", err)
	}
	got, err := box.Decrypt("legacy")
	if err != nil || got != "legacy" {
		t.Fatalf("legacy decrypt = %q %v", got, err)
	}
}
