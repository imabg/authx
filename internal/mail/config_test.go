package mail

import (
	"strings"
	"testing"
)

func TestConfigPublicMasksSecrets(t *testing.T) {
	cfg := Config{
		Provider:  ProviderSendGrid,
		FromEmail: "noreply@example.com",
		SendGrid:  SendGridConfig{APIKey: "sg-secret"},
		SMTP:      SMTPConfig{Username: "acme", Password: "smtp-secret"},
	}
	got := cfg.Public()
	if got.SendGrid.APIKey != MaskedSecret {
		t.Fatalf("api key = %q", got.SendGrid.APIKey)
	}
	if got.SMTP.Username != "" || got.SMTP.Password != "" {
		t.Fatalf("smtp credentials leaked: %+v", got.SMTP)
	}
	if cfg.SendGrid.APIKey != "sg-secret" || cfg.SMTP.Username != "acme" || cfg.SMTP.Password != "smtp-secret" {
		t.Fatal("Public mutated the original config")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{name: "log is valid", cfg: Config{Provider: ProviderLog}},
		{name: "empty provider is valid", cfg: Config{}},
		{
			name:    "unknown provider",
			cfg:     Config{Provider: "mailgun"},
			wantErr: "mail.provider must be log, sendgrid, or smtp",
		},
		{
			name:    "sendgrid missing from",
			cfg:     Config{Provider: ProviderSendGrid, SendGrid: SendGridConfig{APIKey: "sg"}},
			wantErr: "mail.from_email is required for sendgrid",
		},
		{
			name:    "sendgrid missing key",
			cfg:     Config{Provider: ProviderSendGrid, FromEmail: "a@b.com"},
			wantErr: "mail.sendgrid.api_key is required",
		},
		{
			name: "sendgrid valid",
			cfg:  Config{Provider: ProviderSendGrid, FromEmail: "a@b.com", SendGrid: SendGridConfig{APIKey: "sg"}},
		},
		{
			name:    "smtp missing host",
			cfg:     Config{Provider: ProviderSMTP, FromEmail: "a@b.com", SMTP: SMTPConfig{Port: 587}},
			wantErr: "mail.smtp.host is required",
		},
		{
			name:    "smtp bad port",
			cfg:     Config{Provider: ProviderSMTP, FromEmail: "a@b.com", SMTP: SMTPConfig{Host: "smtp.example.com", Port: 0}},
			wantErr: "mail.smtp.port must be between 1 and 65535",
		},
		{
			name: "smtp valid",
			cfg: Config{
				Provider:  ProviderSMTP,
				FromEmail: "a@b.com",
				SMTP:      SMTPConfig{Host: "smtp.example.com", Port: 587, TLS: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestSecretUnchanged(t *testing.T) {
	if !SecretUnchanged("") || !SecretUnchanged(MaskedSecret) {
		t.Fatal("expected empty and masked secrets to be unchanged")
	}
	if SecretUnchanged("new-key") {
		t.Fatal("real secret should be treated as a change")
	}
}
