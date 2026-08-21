package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/imabg/authx/internal/mail"
	"github.com/imabg/authx/internal/validate"
)

type AuthMethod string

const (
	AuthMethodPassword  AuthMethod = "password"
	AuthMethodOTP       AuthMethod = "otp"
	AuthMethodMagicLink AuthMethod = "magic_link"
)

type PasswordSettings struct {
	MinLength        int  `json:"min_length" validate:"gte=6"`
	RequireMixedCase bool `json:"require_mixed_case"`
	RequireDigit     bool `json:"require_digit"`
}

type OTPSettings struct {
	Length      int `json:"length" validate:"gte=4,lte=12"`
	TTLSeconds  int `json:"ttl_seconds" validate:"gte=30,lte=86400"`
	MaxAttempts int `json:"max_attempts" validate:"gte=1"`
}

type MagicLinkSettings struct {
	TTLSeconds int `json:"ttl_seconds" validate:"gte=30"`
}

type TokenSettings struct {
	AccessTTLSeconds  int `json:"access_ttl_seconds"`
	RefreshTTLSeconds int `json:"refresh_ttl_seconds"`
}

type Settings struct {
	SignupEnabled             bool              `json:"signup_enabled"`
	AuthMethod                AuthMethod        `json:"auth_method" validate:"required,oneof=password otp magic_link"`
	Password                  PasswordSettings  `json:"password"`
	OTP                       OTPSettings       `json:"otp"`
	MagicLink                 MagicLinkSettings `json:"magic_link"`
	Tokens                    TokenSettings     `json:"tokens"`
	EmailVerificationRequired bool              `json:"email_verification_required"`
	BlockedDomains            []string          `json:"blocked_domains"`
	Mail                      mail.Config       `json:"mail"`
}

func DefaultSettings() Settings {
	return Settings{
		SignupEnabled: true,
		AuthMethod:    AuthMethodPassword,
		Password: PasswordSettings{
			MinLength:        8,
			RequireMixedCase: true,
			RequireDigit:     true,
		},
		OTP: OTPSettings{
			Length:      6,
			TTLSeconds:  300,
			MaxAttempts: 5,
		},
		MagicLink: MagicLinkSettings{
			TTLSeconds: 900,
		},
		Tokens: TokenSettings{
			AccessTTLSeconds:  900,
			RefreshTTLSeconds: 2592000,
		},
		EmailVerificationRequired: false,
		BlockedDomains:            []string{},
		Mail: mail.Config{
			Provider: mail.ProviderLog,
		},
	}
}

func DecodeSettings(raw []byte) (Settings, error) {
	settings := DefaultSettings()
	if len(raw) == 0 || string(raw) == "null" {
		if err := prepareSettings(&settings); err != nil {
			return Settings{}, err
		}
		return settings, nil
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return Settings{}, fmt.Errorf("decode application settings: %w", err)
	}
	if err := prepareSettings(&settings); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func MergeSettings(base Settings, raw json.RawMessage) (Settings, error) {
	if len(raw) == 0 || string(raw) == "null" {
		if err := prepareSettings(&base); err != nil {
			return Settings{}, err
		}
		return base, nil
	}
	var patch settingsPatch
	if err := json.Unmarshal(raw, &patch); err != nil {
		return Settings{}, fmt.Errorf("decode application settings: %w", err)
	}
	merged, err := applySettingsPatch(base, patch)
	if err != nil {
		return Settings{}, err
	}
	if err := prepareSettings(&merged); err != nil {
		return Settings{}, err
	}
	return merged, nil
}

func (s Settings) Public() Settings {
	out := s
	out.Mail = s.Mail.Public()
	return out
}

func applySettingDefaults(settings *Settings) {
	defaults := DefaultSettings()
	if settings.Password.MinLength <= 0 {
		settings.Password.MinLength = defaults.Password.MinLength
	}
	if settings.OTP.Length <= 0 {
		settings.OTP.Length = defaults.OTP.Length
	}
	if settings.OTP.TTLSeconds <= 0 {
		settings.OTP.TTLSeconds = defaults.OTP.TTLSeconds
	}
	if settings.OTP.MaxAttempts <= 0 {
		settings.OTP.MaxAttempts = defaults.OTP.MaxAttempts
	}
	if settings.MagicLink.TTLSeconds <= 0 {
		settings.MagicLink.TTLSeconds = defaults.MagicLink.TTLSeconds
	}
	if settings.Tokens.AccessTTLSeconds <= 0 {
		settings.Tokens.AccessTTLSeconds = defaults.Tokens.AccessTTLSeconds
	}
	if settings.Tokens.RefreshTTLSeconds <= 0 {
		settings.Tokens.RefreshTTLSeconds = defaults.Tokens.RefreshTTLSeconds
	}
	if settings.Mail.Provider == "" {
		settings.Mail.Provider = mail.ProviderLog
	}
	if settings.BlockedDomains == nil {
		settings.BlockedDomains = []string{}
	}
}

func prepareSettings(settings *Settings) error {
	applySettingDefaults(settings)
	domains, err := NormalizeBlockedDomains(settings.BlockedDomains)
	if err != nil {
		return err
	}
	settings.BlockedDomains = domains
	return ValidateSettings(*settings)
}

// NormalizeBlockedDomains lowercases and trims each domain, rejects empty entries
// and values that include '@', and de-duplicates while preserving order.
func NormalizeBlockedDomains(domains []string) ([]string, error) {
	if len(domains) == 0 {
		return []string{}, nil
	}
	out := make([]string, 0, len(domains))
	seen := make(map[string]struct{}, len(domains))
	for _, raw := range domains {
		domain := strings.ToLower(strings.TrimSpace(raw))
		domain = strings.TrimPrefix(domain, ".")
		domain = strings.TrimSuffix(domain, ".")
		if domain == "" {
			return nil, fmt.Errorf("blocked_domains entries must not be empty")
		}
		if strings.Contains(domain, "@") {
			return nil, fmt.Errorf("blocked_domains must not include @ prefix")
		}
		if !validBlockedDomain(domain) {
			return nil, fmt.Errorf("blocked_domains contains an invalid domain")
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		out = append(out, domain)
	}
	return out, nil
}

func validBlockedDomain(domain string) bool {
	if domain == "" || strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") {
		return false
	}
	labelStart := true
	lastWasDot := false
	for _, r := range domain {
		switch {
		case r == '.':
			if labelStart || lastWasDot {
				return false
			}
			lastWasDot = true
			labelStart = true
		case r == '-':
			if labelStart {
				return false
			}
			lastWasDot = false
			labelStart = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			lastWasDot = false
			labelStart = false
		default:
			return false
		}
	}
	return !labelStart && !lastWasDot
}

func EmailDomain(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return ""
	}
	return email[at+1:]
}

func (s Settings) EmailDomainBlocked(email string) bool {
	domain := EmailDomain(email)
	if domain == "" {
		return false
	}
	for _, blocked := range s.BlockedDomains {
		if domain == blocked {
			return true
		}
	}
	return false
}

func ValidateSettings(settings Settings) error {
	if err := validate.Struct(settings); err != nil {
		return mapSettingsValidationError(err)
	}
	if _, err := NormalizeBlockedDomains(settings.BlockedDomains); err != nil {
		return err
	}
	return nil
}

func ParseAuthMethod(v string) (AuthMethod, error) {
	method := AuthMethod(strings.ToLower(strings.TrimSpace(v)))
	switch method {
	case AuthMethodPassword, AuthMethodOTP, AuthMethodMagicLink:
		return method, nil
	default:
		return "", fmt.Errorf("auth_method must be password, otp, or magic_link")
	}
}

type settingsPatch struct {
	SignupEnabled             *bool           `json:"signup_enabled"`
	AuthMethod                *AuthMethod     `json:"auth_method"`
	Password                  *passwordPatch  `json:"password"`
	OTP                       *otpPatch       `json:"otp"`
	MagicLink                 *magicLinkPatch `json:"magic_link"`
	Tokens                    *tokenPatch     `json:"tokens"`
	EmailVerificationRequired *bool           `json:"email_verification_required"`
	BlockedDomains            *[]string       `json:"blocked_domains"`
	Mail                      *mailPatch      `json:"mail"`
}

type passwordPatch struct {
	MinLength        *int  `json:"min_length"`
	RequireMixedCase *bool `json:"require_mixed_case"`
	RequireDigit     *bool `json:"require_digit"`
}

type otpPatch struct {
	Length      *int `json:"length"`
	TTLSeconds  *int `json:"ttl_seconds"`
	MaxAttempts *int `json:"max_attempts"`
}

type magicLinkPatch struct {
	TTLSeconds *int `json:"ttl_seconds"`
}

type tokenPatch struct {
	AccessTTLSeconds  *int `json:"access_ttl_seconds"`
	RefreshTTLSeconds *int `json:"refresh_ttl_seconds"`
}

type mailPatch struct {
	Provider  *mail.Provider `json:"provider"`
	FromEmail *string        `json:"from_email"`
	FromName  *string        `json:"from_name"`
	SendGrid  *sendGridPatch `json:"sendgrid"`
	SMTP      *smtpPatch     `json:"smtp"`
}

type sendGridPatch struct {
	APIKey *string `json:"api_key"`
}

type smtpPatch struct {
	Host       *string `json:"host"`
	Port       *int    `json:"port"`
	Username   *string `json:"username"`
	Password   *string `json:"password"`
	TLS        *bool   `json:"tls"`
	SkipVerify *bool   `json:"skip_verify"`
}

func applySettingsPatch(base Settings, patch settingsPatch) (Settings, error) {
	if patch.SignupEnabled != nil {
		base.SignupEnabled = *patch.SignupEnabled
	}
	if patch.AuthMethod != nil {
		base.AuthMethod = *patch.AuthMethod
	}
	if patch.EmailVerificationRequired != nil {
		base.EmailVerificationRequired = *patch.EmailVerificationRequired
	}
	if patch.BlockedDomains != nil {
		base.BlockedDomains = *patch.BlockedDomains
	}
	if patch.Password != nil {
		if patch.Password.MinLength != nil {
			base.Password.MinLength = *patch.Password.MinLength
		}
		if patch.Password.RequireMixedCase != nil {
			base.Password.RequireMixedCase = *patch.Password.RequireMixedCase
		}
		if patch.Password.RequireDigit != nil {
			base.Password.RequireDigit = *patch.Password.RequireDigit
		}
	}
	if patch.OTP != nil {
		if patch.OTP.Length != nil {
			base.OTP.Length = *patch.OTP.Length
		}
		if patch.OTP.TTLSeconds != nil {
			base.OTP.TTLSeconds = *patch.OTP.TTLSeconds
		}
		if patch.OTP.MaxAttempts != nil {
			base.OTP.MaxAttempts = *patch.OTP.MaxAttempts
		}
	}
	if patch.MagicLink != nil {
		if patch.MagicLink.TTLSeconds != nil {
			base.MagicLink.TTLSeconds = *patch.MagicLink.TTLSeconds
		}
	}
	if patch.Tokens != nil {
		if patch.Tokens.AccessTTLSeconds != nil {
			base.Tokens.AccessTTLSeconds = *patch.Tokens.AccessTTLSeconds
		}
		if patch.Tokens.RefreshTTLSeconds != nil {
			base.Tokens.RefreshTTLSeconds = *patch.Tokens.RefreshTTLSeconds
		}
	}
	if patch.Mail != nil {
		updated, err := applyMailPatch(base.Mail, *patch.Mail)
		if err != nil {
			return Settings{}, err
		}
		base.Mail = updated
	}
	return base, nil
}

func applyMailPatch(base mail.Config, patch mailPatch) (mail.Config, error) {
	if patch.Provider != nil {
		base.Provider = *patch.Provider
	}
	if patch.FromEmail != nil {
		base.FromEmail = strings.TrimSpace(*patch.FromEmail)
	}
	if patch.FromName != nil {
		base.FromName = strings.TrimSpace(*patch.FromName)
	}
	if patch.SendGrid != nil && patch.SendGrid.APIKey != nil && !mail.SecretUnchanged(*patch.SendGrid.APIKey) {
		base.SendGrid.APIKey = *patch.SendGrid.APIKey
	}
	if patch.SMTP != nil {
		if patch.SMTP.Host != nil {
			base.SMTP.Host = strings.TrimSpace(*patch.SMTP.Host)
		}
		if patch.SMTP.Port != nil {
			base.SMTP.Port = *patch.SMTP.Port
		}
		if patch.SMTP.Username != nil {
			base.SMTP.Username = strings.TrimSpace(*patch.SMTP.Username)
		}
		if patch.SMTP.Password != nil && !mail.SecretUnchanged(*patch.SMTP.Password) {
			decoded, err := mail.DecodeSMTPPassword(*patch.SMTP.Password)
			if err != nil {
				return mail.Config{}, err
			}
			base.SMTP.Password = decoded
		}
		if patch.SMTP.TLS != nil {
			base.SMTP.TLS = *patch.SMTP.TLS
		}
		if patch.SMTP.SkipVerify != nil {
			base.SMTP.SkipVerify = *patch.SMTP.SkipVerify
		}
	}
	return base, nil
}
