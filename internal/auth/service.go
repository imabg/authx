package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/challenge"
	"github.com/imabg/authx/internal/hasher"
	"github.com/imabg/authx/internal/mail"
	"github.com/imabg/authx/internal/token"
	"github.com/imabg/authx/internal/users"
)

type Service struct {
	users      users.IUserRepository
	challenges challenge.IService
	tokens     token.IService
	mailer     mail.Mailer
	limiter    *RateLimiter
	baseURL    string
}

func NewService(usersRepo users.IUserRepository, challenges challenge.IService, tokens token.IService, mailer mail.Mailer, publicBaseURL string) *Service {
	return &Service{
		users:      usersRepo,
		challenges: challenges,
		tokens:     tokens,
		mailer:     mailer,
		limiter:    NewRateLimiter(5, 15*time.Minute),
		baseURL:    publicBaseURL,
	}
}

func (s *Service) Authenticate(ctx context.Context, application *app.Application, req Request) (*Result, error) {
	req.normalize()
	if err := rejectDisallowedEmail(application, req.Email); err != nil {
		return nil, err
	}
	switch application.Settings.AuthMethod {
	case app.AuthMethodPassword:
		return s.authenticatePassword(ctx, application, req)
	case app.AuthMethodOTP:
		return s.authenticateOTP(ctx, application, req)
	case app.AuthMethodMagicLink:
		return s.authenticateMagicLink(ctx, application, req)
	default:
		return nil, fmt.Errorf("unsupported auth_method")
	}
}

func (s *Service) Refresh(ctx context.Context, application *app.Application, refreshToken string) (*Result, error) {
	if refreshToken == "" {
		return nil, ErrInvalidCredentials
	}
	rec, err := s.tokens.LookupRefresh(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if rec.ApplicationID != application.ID {
		return nil, ErrInvalidCredentials
	}
	user, err := s.users.GetByID(ctx, rec.UserID)
	if err != nil || !user.Active() {
		return nil, ErrInvalidCredentials
	}
	if err := rejectDisallowedEmail(application, user.Email); err != nil {
		return nil, err
	}
	pair, err := s.tokens.Rotate(ctx, refreshToken, application.ID, user, application)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	return authenticatedResult(pair, user), nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return ErrValidation
	}
	return s.tokens.Revoke(ctx, refreshToken)
}

func (s *Service) authenticatePassword(ctx context.Context, application *app.Application, req Request) (*Result, error) {
	if req.Email == "" || req.Password == "" {
		return nil, ErrInvalidPayload
	}
	if req.Code != "" && !application.Settings.EmailVerificationRequired {
		return nil, ErrInvalidPayload
	}
	if !LooksLikeEmail(req.Email) {
		return nil, ErrValidation
	}

	user, err := s.users.GetByEmail(ctx, application.ID, req.Email)
	if err != nil {
		if !errors.Is(err, users.ErrNotFound) {
			return nil, err
		}
		if !application.Settings.SignupEnabled {
			return nil, ErrInvalidCredentials
		}
		if err := ValidatePassword(req.Password, PasswordPolicy{
			MinLength:        application.Settings.Password.MinLength,
			RequireMixedCase: application.Settings.Password.RequireMixedCase,
			RequireDigit:     application.Settings.Password.RequireDigit,
		}); err != nil {
			return nil, err
		}
		hash, err := hasher.Hash(req.Password)
		if err != nil {
			return nil, err
		}
		created, err := s.users.CreateWithCredential(ctx, users.User{
			ApplicationID: application.ID,
			Email:         req.Email,
			FirstName:     req.FirstName,
			LastName:      req.LastName,
			SignupMethod:  string(app.AuthMethodPassword),
			Status:        users.StatusActive,
		}, hash)
		if err != nil {
			return nil, err
		}
		user = created
	} else {
		if !user.Active() {
			return nil, ErrInvalidCredentials
		}
		hash, err := s.users.GetPasswordHash(ctx, user.ID)
		if err != nil {
			return nil, ErrInvalidCredentials
		}
		ok, err := hasher.Compare(req.Password, hash)
		if err != nil || !ok {
			return nil, ErrInvalidCredentials
		}
	}

	if application.Settings.EmailVerificationRequired && user.EmailVerifiedAt == nil {
		if req.Code == "" {
			return s.sendOTPChallenge(ctx, application, &user.ID, req.Email, req.IP)
		}
		if _, err := s.challenges.VerifyOTP(ctx, application.ID, req.Email, req.Code, application.Settings.OTP.MaxAttempts); err != nil {
			return nil, ErrInvalidCredentials
		}
		now := time.Now()
		user.EmailVerifiedAt = &now
		updated, err := s.users.Update(ctx, user)
		if err != nil {
			return nil, err
		}
		user = updated
	}

	return s.issue(ctx, application, user)
}

func (s *Service) authenticateOTP(ctx context.Context, application *app.Application, req Request) (*Result, error) {
	if req.Password != "" || req.Token != "" {
		return nil, ErrInvalidPayload
	}
	if req.Email == "" {
		return nil, ErrInvalidPayload
	}
	if !LooksLikeEmail(req.Email) {
		return nil, ErrValidation
	}
	if req.Code == "" {
		return s.sendOTPChallenge(ctx, application, nil, req.Email, req.IP)
	}
	if _, err := s.challenges.VerifyOTP(ctx, application.ID, req.Email, req.Code, application.Settings.OTP.MaxAttempts); err != nil {
		return nil, ErrInvalidCredentials
	}
	user, err := s.upsertUser(ctx, application, req, string(app.AuthMethodOTP), true)
	if err != nil {
		return nil, err
	}
	return s.issue(ctx, application, user)
}

func (s *Service) authenticateMagicLink(ctx context.Context, application *app.Application, req Request) (*Result, error) {
	if req.Password != "" || req.Code != "" {
		return nil, ErrInvalidPayload
	}
	if req.Token == "" {
		if req.Email == "" {
			return nil, ErrInvalidPayload
		}
		if !LooksLikeEmail(req.Email) {
			return nil, ErrValidation
		}
		return s.sendMagicLinkChallenge(ctx, application, req.Email, req.IP)
	}
	c, err := s.challenges.ConsumeMagicLink(ctx, application.ID, req.Token, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if req.Email == "" {
		req.Email = c.Target
	}
	if err := rejectDisallowedEmail(application, req.Email); err != nil {
		return nil, err
	}
	user, err := s.upsertUser(ctx, application, req, string(app.AuthMethodMagicLink), true)
	if err != nil {
		return nil, err
	}
	return s.issue(ctx, application, user)
}

func (s *Service) sendOTPChallenge(ctx context.Context, application *app.Application, userID *uuid.UUID, email, ip string) (*Result, error) {
	if !s.limiter.Allow(rateKey(application.ID, email, ip)) {
		return nil, ErrRateLimited
	}
	existing, err := s.users.GetByEmail(ctx, application.ID, email)
	shouldSend := true
	if errors.Is(err, users.ErrNotFound) {
		if !application.Settings.SignupEnabled {
			shouldSend = false
		}
	} else if err != nil {
		return nil, err
	} else {
		userID = &existing.ID
		if !existing.Active() {
			shouldSend = false
		}
	}
	ttl := time.Duration(application.Settings.OTP.TTLSeconds) * time.Second
	if !shouldSend {
		return challengeResult(challenge.TypeOTP, ttl), nil
	}
	code, issuedTTL, err := s.challenges.IssueOTP(ctx, application.ID, userID, email, application.Settings.OTP.Length, application.Settings.OTP.TTLSeconds)
	if err != nil {
		return nil, err
	}
	if err := s.mailer.SendOTP(ctx, application.Settings.Mail, email, code); err != nil {
		return nil, err
	}
	return challengeResult(challenge.TypeOTP, issuedTTL), nil
}

func (s *Service) sendMagicLinkChallenge(ctx context.Context, application *app.Application, email, ip string) (*Result, error) {
	if !s.limiter.Allow(rateKey(application.ID, email, ip)) {
		return nil, ErrRateLimited
	}
	existing, err := s.users.GetByEmail(ctx, application.ID, email)
	shouldSend := true
	var userID *uuid.UUID
	if errors.Is(err, users.ErrNotFound) {
		if !application.Settings.SignupEnabled {
			shouldSend = false
		}
	} else if err != nil {
		return nil, err
	} else {
		userID = &existing.ID
		if !existing.Active() {
			shouldSend = false
		}
	}
	ttl := time.Duration(application.Settings.MagicLink.TTLSeconds) * time.Second
	if !shouldSend {
		return challengeResult(challenge.TypeMagicLink, ttl), nil
	}
	tokenValue, issuedTTL, err := s.challenges.IssueMagicLink(ctx, application.ID, userID, email, application.Settings.MagicLink.TTLSeconds)
	if err != nil {
		return nil, err
	}
	link := magicLinkURL(s.baseURL, tokenValue, email)
	if err := s.mailer.SendMagicLink(ctx, application.Settings.Mail, email, link); err != nil {
		return nil, err
	}
	return challengeResult(challenge.TypeMagicLink, issuedTTL), nil
}

func (s *Service) upsertUser(ctx context.Context, application *app.Application, req Request, signupMethod string, verified bool) (users.User, error) {
	user, err := s.users.GetByEmail(ctx, application.ID, req.Email)
	if err == nil {
		if !user.Active() {
			return users.User{}, ErrInvalidCredentials
		}
		if verified && user.EmailVerifiedAt == nil {
			now := time.Now()
			user.EmailVerifiedAt = &now
			return s.users.Update(ctx, user)
		}
		return user, nil
	}
	if !errors.Is(err, users.ErrNotFound) {
		return users.User{}, err
	}
	if !application.Settings.SignupEnabled {
		return users.User{}, ErrInvalidCredentials
	}
	var verifiedAt *time.Time
	if verified {
		now := time.Now()
		verifiedAt = &now
	}
	return s.users.Create(ctx, users.User{
		ApplicationID:   application.ID,
		Email:           req.Email,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		SignupMethod:    signupMethod,
		EmailVerifiedAt: verifiedAt,
		Status:          users.StatusActive,
	})
}

func (s *Service) issue(ctx context.Context, application *app.Application, user users.User) (*Result, error) {
	pair, err := s.tokens.Issue(ctx, user, application)
	if err != nil {
		return nil, err
	}
	return authenticatedResult(pair, user), nil
}

func authenticatedResult(pair token.Pair, user users.User) *Result {
	return &Result{
		Status:       StatusAuthenticated,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
		TokenType:    pair.TokenType,
		User:         &user,
	}
}

func rejectDisallowedEmail(application *app.Application, email string) error {
	if email == "" {
		return nil
	}
	if !LooksLikeEmail(email) {
		return ErrValidation
	}
	if application != nil && application.Settings.EmailDomainBlocked(email) {
		return ErrEmailDomainBlocked
	}
	return nil
}

func rateKey(applicationID uuid.UUID, email, ip string) string {
	return applicationID.String() + "|" + email + "|" + ip
}
