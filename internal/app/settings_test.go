package app_test

import (
	"strings"
	"testing"

	"github.com/imabg/authx/internal/app"
)

func TestDecodeSettings(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
		check   func(*testing.T, app.Settings)
	}{
		{
			name: "applies defaults for otp",
			raw:  `{"auth_method":"otp","signup_enabled":false}`,
			check: func(t *testing.T, settings app.Settings) {
				t.Helper()
				if settings.AuthMethod != app.AuthMethodOTP {
					t.Fatalf("auth_method = %s", settings.AuthMethod)
				}
				if settings.SignupEnabled {
					t.Fatal("expected signup_enabled false")
				}
				if settings.OTP.Length != 6 || settings.Tokens.AccessTTLSeconds != 900 {
					t.Fatalf("defaults not applied: %+v", settings)
				}
			},
		},
		{
			name: "empty payload uses defaults",
			raw:  "",
			check: func(t *testing.T, settings app.Settings) {
				t.Helper()
				if settings.AuthMethod != app.AuthMethodPassword || !settings.SignupEnabled {
					t.Fatalf("defaults = %+v", settings)
				}
			},
		},
		{
			name:    "rejects unknown method",
			raw:     `{"auth_method":"saml"}`,
			wantErr: "auth_method must be password, otp, or magic_link",
		},
		{
			name:    "rejects invalid json",
			raw:     `{`,
			wantErr: "decode application settings",
		},
		{
			name:    "rejects short otp ttl",
			raw:     `{"auth_method":"otp","otp":{"length":6,"ttl_seconds":5}}`,
			wantErr: "otp.ttl_seconds must be at least 30",
		},
		{
			name: "accepts sendgrid mail settings",
			raw:  `{"auth_method":"otp","mail":{"provider":"sendgrid","from_email":"noreply@example.com","from_name":"Acme","sendgrid":{"api_key":"sg-secret"}}}`,
			check: func(t *testing.T, settings app.Settings) {
				t.Helper()
				if settings.Mail.Provider != "sendgrid" || settings.Mail.SendGrid.APIKey != "sg-secret" {
					t.Fatalf("mail = %+v", settings.Mail)
				}
			},
		},
		{
			name: "normalizes blocked_domains",
			raw:  `{"auth_method":"password","blocked_domains":[" Blocked.COM ", "evil.org", "blocked.com"]}`,
			check: func(t *testing.T, settings app.Settings) {
				t.Helper()
				if len(settings.BlockedDomains) != 2 || settings.BlockedDomains[0] != "blocked.com" || settings.BlockedDomains[1] != "evil.org" {
					t.Fatalf("blocked_domains = %#v", settings.BlockedDomains)
				}
			},
		},
		{
			name:    "rejects empty blocked domain",
			raw:     `{"auth_method":"password","blocked_domains":[" "]}`,
			wantErr: "blocked_domains entries must not be empty",
		},
		{
			name:    "rejects blocked domain with @",
			raw:     `{"auth_method":"password","blocked_domains":["@blocked.com"]}`,
			wantErr: "blocked_domains must not include @ prefix",
		},
		{
			name:    "rejects sendgrid without api key",
			raw:     `{"auth_method":"otp","mail":{"provider":"sendgrid","from_email":"noreply@example.com"}}`,
			wantErr: "mail.sendgrid.api_key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings, err := app.DecodeSettings([]byte(tt.raw))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeSettings: %v", err)
			}
			tt.check(t, settings)
		})
	}
}

func TestValidateAndParseAuthMethod(t *testing.T) {
	tests := []struct {
		name    string
		run     func() error
		wantErr string
	}{
		{
			name: "valid password settings",
			run:  func() error { return app.ValidateSettings(app.DefaultSettings()) },
		},
		{
			name: "password min length too small",
			run: func() error {
				s := app.DefaultSettings()
				s.Password.MinLength = 4
				return app.ValidateSettings(s)
			},
			wantErr: "password.min_length must be at least 6",
		},
		{
			name: "parse magic_link",
			run: func() error {
				_, err := app.ParseAuthMethod(" Magic_Link ")
				return err
			},
		},
		{
			name: "parse unknown method",
			run: func() error {
				_, err := app.ParseAuthMethod("saml")
				return err
			},
			wantErr: "auth_method must be password, otp, or magic_link",
		},
		{
			name: "otp ttl too small",
			run: func() error {
				s := app.DefaultSettings()
				s.OTP.TTLSeconds = 10
				return app.ValidateSettings(s)
			},
			wantErr: "otp.ttl_seconds must be at least 30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMergeSettings(t *testing.T) {
	base := app.DefaultSettings()
	base.OTP.Length = 6
	base.OTP.TTLSeconds = 300
	base.Mail.Provider = "sendgrid"
	base.Mail.FromEmail = "noreply@example.com"
	base.Mail.SendGrid.APIKey = "sg-real-key"

	t.Run("patches otp length without dropping ttl or mail", func(t *testing.T) {
		got, err := app.MergeSettings(base, []byte(`{"otp":{"length":8}}`))
		if err != nil {
			t.Fatalf("MergeSettings: %v", err)
		}
		if got.OTP.Length != 8 || got.OTP.TTLSeconds != 300 {
			t.Fatalf("otp = %+v", got.OTP)
		}
		if got.Mail.SendGrid.APIKey != "sg-real-key" {
			t.Fatalf("api key overwritten: %q", got.Mail.SendGrid.APIKey)
		}
	})

	t.Run("keeps secret when masked value is posted back", func(t *testing.T) {
		got, err := app.MergeSettings(base, []byte(`{"mail":{"provider":"sendgrid","from_email":"noreply@example.com","sendgrid":{"api_key":"********"}}}`))
		if err != nil {
			t.Fatalf("MergeSettings: %v", err)
		}
		if got.Mail.SendGrid.APIKey != "sg-real-key" {
			t.Fatalf("api key = %q", got.Mail.SendGrid.APIKey)
		}
	})

	t.Run("updates smtp settings", func(t *testing.T) {
		got, err := app.MergeSettings(base, []byte(`{"mail":{"provider":"smtp","from_email":"noreply@example.com","from_name":"Acme","smtp":{"host":"smtp.example.com","port":587,"username":"acme","password":"s3cret","tls":true}}}`))
		if err != nil {
			t.Fatalf("MergeSettings: %v", err)
		}
		if got.Mail.Provider != "smtp" || got.Mail.SMTP.Host != "smtp.example.com" || got.Mail.SMTP.Password != "s3cret" {
			t.Fatalf("mail = %+v", got.Mail)
		}
	})

	t.Run("rejects zero otp length after merge", func(t *testing.T) {
		_, err := app.MergeSettings(base, []byte(`{"otp":{"length":3}}`))
		if err == nil || !strings.Contains(err.Error(), "otp.length") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("replaces blocked_domains when sent", func(t *testing.T) {
		base.BlockedDomains = []string{"old.com"}
		got, err := app.MergeSettings(base, []byte(`{"blocked_domains":[" New.ORG ", "new.org"]}`))
		if err != nil {
			t.Fatalf("MergeSettings: %v", err)
		}
		if len(got.BlockedDomains) != 1 || got.BlockedDomains[0] != "new.org" {
			t.Fatalf("blocked_domains = %#v", got.BlockedDomains)
		}
	})

	t.Run("keeps blocked_domains when omitted", func(t *testing.T) {
		base.BlockedDomains = []string{"old.com"}
		got, err := app.MergeSettings(base, []byte(`{"otp":{"length":8}}`))
		if err != nil {
			t.Fatalf("MergeSettings: %v", err)
		}
		if len(got.BlockedDomains) != 1 || got.BlockedDomains[0] != "old.com" {
			t.Fatalf("blocked_domains = %#v", got.BlockedDomains)
		}
	})

	t.Run("clears blocked_domains with empty list", func(t *testing.T) {
		base.BlockedDomains = []string{"old.com"}
		got, err := app.MergeSettings(base, []byte(`{"blocked_domains":[]}`))
		if err != nil {
			t.Fatalf("MergeSettings: %v", err)
		}
		if len(got.BlockedDomains) != 0 {
			t.Fatalf("blocked_domains = %#v", got.BlockedDomains)
		}
	})
}

func TestSettingsPublicMasksMailSecrets(t *testing.T) {
	settings := app.DefaultSettings()
	settings.Mail.SendGrid.APIKey = "sg-real"
	settings.Mail.SMTP.Password = "smtp-real"
	got := settings.Public()
	if got.Mail.SendGrid.APIKey != "********" || got.Mail.SMTP.Password != "********" {
		t.Fatalf("public mail = %+v", got.Mail)
	}
	if settings.Mail.SendGrid.APIKey != "sg-real" {
		t.Fatal("Public mutated original settings")
	}
}

func TestEmailDomainBlocked(t *testing.T) {
	settings := app.DefaultSettings()
	settings.BlockedDomains = []string{"blocked.com", "evil.org"}

	tests := []struct {
		email string
		want  bool
	}{
		{email: "user@blocked.com", want: true},
		{email: "user@Domain.COM", want: false},
		{email: "user@BLOCKED.COM", want: true},
		{email: " user@Evil.ORG ", want: true},
		{email: "user@mail.blocked.com", want: false},
		{email: "user@example.com", want: false},
		{email: "not-an-email", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if got := settings.EmailDomainBlocked(tt.email); got != tt.want {
				t.Fatalf("EmailDomainBlocked(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}

	if app.EmailDomain("User@Domain.COM") != "domain.com" {
		t.Fatalf("EmailDomain = %q", app.EmailDomain("User@Domain.COM"))
	}
}
