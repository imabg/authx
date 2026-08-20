package auth

import (
	"net/url"
	"strings"
	"time"

	"github.com/imabg/authx/internal/users"
	"github.com/imabg/authx/internal/validate"
)

const (
	StatusAuthenticated = "authenticated"
	StatusChallengeSent = "challenge_sent"
)

type Request struct {
	Email     string `validate:"omitempty,authemail"`
	Password  string
	Code      string
	Token     string
	FirstName string `validate:"omitempty,runemax=25"`
	LastName  string `validate:"omitempty,runemax=50"`
	IP        string `validate:"-"`
}

type Result struct {
	Status        string      `json:"status"`
	AccessToken   string      `json:"access_token,omitempty"`
	RefreshToken  string      `json:"refresh_token,omitempty"`
	ExpiresIn     int64       `json:"expires_in,omitempty"`
	TokenType     string      `json:"token_type,omitempty"`
	User          *users.User `json:"user,omitempty"`
	ChallengeType string      `json:"challenge_type,omitempty"`
}

func (r *Request) normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Password = strings.TrimSpace(r.Password)
	r.Code = strings.TrimSpace(r.Code)
	r.Token = strings.TrimSpace(r.Token)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
}

func (r *Request) Validate() error {
	return validate.Map(validate.Struct(r), ErrValidation, func(path, tag, param string) (string, bool) {
		if path == "email" && (tag == "authemail" || tag == "email") {
			return "email is invalid", true
		}
		return validate.StandardMessages(path, tag, param)
	})
}

func challengeResult(challengeType string, ttl time.Duration) *Result {
	return &Result{
		Status:        StatusChallengeSent,
		ChallengeType: challengeType,
		ExpiresIn:     int64(ttl.Seconds()),
	}
}

func magicLinkURL(baseURL, token, email string) string {
	base := strings.TrimRight(baseURL, "/")
	u, err := url.Parse(base + "/auth/callback")
	if err != nil {
		return base + "/auth/callback?token=" + url.QueryEscape(token)
	}
	q := u.Query()
	q.Set("token", token)
	if email != "" {
		q.Set("email", email)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
