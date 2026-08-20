package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/auth"
	"github.com/imabg/authx/internal/challenge"
	challengemock "github.com/imabg/authx/internal/challenge/mock"
	"github.com/imabg/authx/internal/hasher"
	mailmock "github.com/imabg/authx/internal/mail/mock"
	"github.com/imabg/authx/internal/token"
	tokenmock "github.com/imabg/authx/internal/token/mock"
	"github.com/imabg/authx/internal/users"
	usersmock "github.com/imabg/authx/internal/users/mock"
	"go.uber.org/mock/gomock"
)

type authDeps struct {
	users      *usersmock.MockIUserRepository
	challenges *challengemock.MockChallengeService
	tokens     *tokenmock.MockTokenService
	mailer     *mailmock.MockMailer
}

func newAuthService(t *testing.T) (*auth.Service, authDeps) {
	t.Helper()
	ctrl := gomock.NewController(t)
	deps := authDeps{
		users:      usersmock.NewMockIUserRepository(ctrl),
		challenges: challengemock.NewMockChallengeService(ctrl),
		tokens:     tokenmock.NewMockTokenService(ctrl),
		mailer:     mailmock.NewMockMailer(ctrl),
	}
	return auth.NewService(deps.users, deps.challenges, deps.tokens, deps.mailer, "http://localhost:3000"), deps
}

func testApplication(id uuid.UUID, method app.AuthMethod, signup bool) *app.Application {
	settings := app.DefaultSettings()
	settings.AuthMethod = method
	settings.SignupEnabled = signup
	return &app.Application{ID: id, Settings: settings, Status: app.StatusActive}
}

func TestServiceAuthenticate(t *testing.T) {
	ctx := context.Background()
	appID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	email := "user@example.com"
	password := "ValidPass1"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	activeUser := users.User{
		ID:            userID,
		ApplicationID: appID,
		Email:         email,
		Status:        users.StatusActive,
	}
	pair := token.Pair{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 900, TokenType: "Bearer"}
	otpTTL := 5 * time.Minute
	linkTTL := 15 * time.Minute

	tests := []struct {
		name       string
		app        *app.Application
		req        auth.Request
		setup      func(authDeps)
		wantErr    error
		wantStatus string
	}{
		{
			name: "password login success",
			app:  testApplication(appID, app.AuthMethodPassword, true),
			req:  auth.Request{Email: "User@example.com", Password: password},
			setup: func(d authDeps) {
				d.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(activeUser, nil)
				d.users.EXPECT().GetPasswordHash(gomock.Any(), userID).Return(hash, nil)
				d.tokens.EXPECT().Issue(gomock.Any(), activeUser, gomock.Any()).Return(pair, nil)
			},
			wantStatus: auth.StatusAuthenticated,
		},
		{
			name: "password login rejects wrong password",
			app:  testApplication(appID, app.AuthMethodPassword, true),
			req:  auth.Request{Email: email, Password: "WrongPass1"},
			setup: func(d authDeps) {
				d.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(activeUser, nil)
				d.users.EXPECT().GetPasswordHash(gomock.Any(), userID).Return(hash, nil)
			},
			wantErr: auth.ErrInvalidCredentials,
		},
		{
			name: "password signup creates user",
			app:  testApplication(appID, app.AuthMethodPassword, true),
			req:  auth.Request{Email: email, Password: password, FirstName: "Ada"},
			setup: func(d authDeps) {
				d.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(users.User{}, users.ErrNotFound)
				d.users.EXPECT().CreateWithCredential(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, user users.User, _ string) (users.User, error) {
						if user.EmailVerifiedAt != nil {
							t.Fatal("password signup must not set email_verified_at")
						}
						return activeUser, nil
					},
				)
				d.tokens.EXPECT().Issue(gomock.Any(), activeUser, gomock.Any()).Return(pair, nil)
			},
			wantStatus: auth.StatusAuthenticated,
		},
		{
			name: "password signup disabled",
			app:  testApplication(appID, app.AuthMethodPassword, false),
			req:  auth.Request{Email: email, Password: password},
			setup: func(d authDeps) {
				d.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(users.User{}, users.ErrNotFound)
			},
			wantErr: auth.ErrInvalidCredentials,
		},
		{
			name:    "password missing fields",
			app:     testApplication(appID, app.AuthMethodPassword, true),
			req:     auth.Request{Email: email},
			setup:   func(authDeps) {},
			wantErr: auth.ErrInvalidPayload,
		},
		{
			name: "otp send challenge",
			app:  testApplication(appID, app.AuthMethodOTP, true),
			req:  auth.Request{Email: email, IP: "127.0.0.1"},
			setup: func(d authDeps) {
				d.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(users.User{}, users.ErrNotFound)
				d.challenges.EXPECT().IssueOTP(gomock.Any(), appID, gomock.Nil(), email, 6, 300).Return("123456", otpTTL, nil)
				d.mailer.EXPECT().SendOTP(gomock.Any(), gomock.Any(), email, "123456").Return(nil)
			},
			wantStatus: auth.StatusChallengeSent,
		},
		{
			name: "otp verify issues tokens",
			app:  testApplication(appID, app.AuthMethodOTP, true),
			req:  auth.Request{Email: email, Code: "123456"},
			setup: func(d authDeps) {
				d.challenges.EXPECT().VerifyOTP(gomock.Any(), appID, email, "123456", 5).Return(challenge.Challenge{Target: email}, nil)
				d.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(users.User{}, users.ErrNotFound)
				d.users.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, user users.User) (users.User, error) {
						if user.EmailVerifiedAt == nil {
							t.Fatal("otp verify must set email_verified_at")
						}
						out := activeUser
						out.EmailVerifiedAt = user.EmailVerifiedAt
						out.EmailVerified = true
						return out, nil
					},
				)
				d.tokens.EXPECT().Issue(gomock.Any(), gomock.Any(), gomock.Any()).Return(pair, nil)
			},
			wantStatus: auth.StatusAuthenticated,
		},
		{
			name: "otp verify rejects bad code",
			app:  testApplication(appID, app.AuthMethodOTP, true),
			req:  auth.Request{Email: email, Code: "000000"},
			setup: func(d authDeps) {
				d.challenges.EXPECT().VerifyOTP(gomock.Any(), appID, email, "000000", 5).Return(challenge.Challenge{}, challenge.ErrNotFound)
			},
			wantErr: auth.ErrInvalidCredentials,
		},
		{
			name:    "otp rejects password in payload",
			app:     testApplication(appID, app.AuthMethodOTP, true),
			req:     auth.Request{Email: email, Password: password},
			setup:   func(authDeps) {},
			wantErr: auth.ErrInvalidPayload,
		},
		{
			name: "magic link send challenge",
			app:  testApplication(appID, app.AuthMethodMagicLink, true),
			req:  auth.Request{Email: email, IP: "127.0.0.1"},
			setup: func(d authDeps) {
				d.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(users.User{}, users.ErrNotFound)
				d.challenges.EXPECT().IssueMagicLink(gomock.Any(), appID, gomock.Nil(), email, 900).Return("ml-token", linkTTL, nil)
				d.mailer.EXPECT().SendMagicLink(gomock.Any(), gomock.Any(), email, gomock.Any()).Return(nil)
			},
			wantStatus: auth.StatusChallengeSent,
		},
		{
			name: "magic link verify existing user",
			app:  testApplication(appID, app.AuthMethodMagicLink, true),
			req:  auth.Request{Email: email, Token: "ml-token"},
			setup: func(d authDeps) {
				verified := activeUser
				now := time.Now()
				verified.EmailVerifiedAt = &now
				d.challenges.EXPECT().ConsumeMagicLink(gomock.Any(), appID, "ml-token", email).Return(challenge.Challenge{Target: email}, nil)
				d.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(verified, nil)
				d.tokens.EXPECT().Issue(gomock.Any(), verified, gomock.Any()).Return(pair, nil)
			},
			wantStatus: auth.StatusAuthenticated,
		},
		{
			name: "magic link verify marks unverified user verified",
			app:  testApplication(appID, app.AuthMethodMagicLink, true),
			req:  auth.Request{Email: email, Token: "ml-token"},
			setup: func(d authDeps) {
				d.challenges.EXPECT().ConsumeMagicLink(gomock.Any(), appID, "ml-token", email).Return(challenge.Challenge{Target: email}, nil)
				d.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(activeUser, nil)
				d.users.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, user users.User) (users.User, error) {
						if user.EmailVerifiedAt == nil {
							t.Fatal("magic link verify must set email_verified_at")
						}
						return user, nil
					},
				)
				d.tokens.EXPECT().Issue(gomock.Any(), gomock.Any(), gomock.Any()).Return(pair, nil)
			},
			wantStatus: auth.StatusAuthenticated,
		},
		{
			name:    "rejects invalid email format",
			app:     testApplication(appID, app.AuthMethodOTP, true),
			req:     auth.Request{Email: "not-an-email"},
			setup:   func(authDeps) {},
			wantErr: auth.ErrValidation,
		},
		{
			name: "rejects blocked domain on password signup",
			app: func() *app.Application {
				a := testApplication(appID, app.AuthMethodPassword, true)
				a.Settings.BlockedDomains = []string{"blocked.com"}
				return a
			}(),
			req:     auth.Request{Email: "user@Blocked.COM", Password: password},
			setup:   func(authDeps) {},
			wantErr: auth.ErrEmailDomainBlocked,
		},
		{
			name: "rejects blocked domain on otp start",
			app: func() *app.Application {
				a := testApplication(appID, app.AuthMethodOTP, true)
				a.Settings.BlockedDomains = []string{"evil.org"}
				return a
			}(),
			req:     auth.Request{Email: "user@evil.org"},
			setup:   func(authDeps) {},
			wantErr: auth.ErrEmailDomainBlocked,
		},
		{
			name: "rejects blocked domain on password login",
			app: func() *app.Application {
				a := testApplication(appID, app.AuthMethodPassword, true)
				a.Settings.BlockedDomains = []string{"blocked.com"}
				return a
			}(),
			req:     auth.Request{Email: "user@blocked.com", Password: password},
			setup:   func(authDeps) {},
			wantErr: auth.ErrEmailDomainBlocked,
		},
		{
			name:    "unsupported auth method",
			app:     &app.Application{ID: appID, Settings: app.Settings{AuthMethod: "saml"}},
			req:     auth.Request{Email: email},
			setup:   func(authDeps) {},
			wantErr: errors.New("unsupported auth_method"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, deps := newAuthService(t)
			tt.setup(deps)
			got, err := svc.Authenticate(ctx, tt.app, tt.req)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v", tt.wantErr)
				}
				if tt.name == "unsupported auth method" {
					if err.Error() != tt.wantErr.Error() {
						t.Fatalf("error = %v, want %v", err, tt.wantErr)
					}
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Authenticate: %v", err)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", got.Status, tt.wantStatus)
			}
		})
	}
}

func TestServiceRefreshAndLogout(t *testing.T) {
	ctx := context.Background()
	appID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	application := testApplication(appID, app.AuthMethodPassword, true)
	user := users.User{ID: userID, ApplicationID: appID, Email: "user@example.com", Status: users.StatusActive}
	pair := token.Pair{AccessToken: "access", RefreshToken: "next", ExpiresIn: 900, TokenType: "Bearer"}

	tests := []struct {
		name    string
		run     func(*auth.Service, authDeps) error
		wantErr error
	}{
		{
			name: "refresh success",
			run: func(svc *auth.Service, d authDeps) error {
				d.tokens.EXPECT().LookupRefresh(gomock.Any(), "rt").Return(token.RefreshRecord{UserID: userID, ApplicationID: appID}, nil)
				d.users.EXPECT().GetByID(gomock.Any(), userID).Return(user, nil)
				d.tokens.EXPECT().Rotate(gomock.Any(), "rt", appID, user, gomock.Any()).Return(pair, nil)
				_, err := svc.Refresh(ctx, application, "rt")
				return err
			},
		},
		{
			name: "refresh blocked domain",
			run: func(svc *auth.Service, d authDeps) error {
				blockedApp := testApplication(appID, app.AuthMethodPassword, true)
				blockedApp.Settings.BlockedDomains = []string{"example.com"}
				d.tokens.EXPECT().LookupRefresh(gomock.Any(), "rt").Return(token.RefreshRecord{UserID: userID, ApplicationID: appID}, nil)
				d.users.EXPECT().GetByID(gomock.Any(), userID).Return(user, nil)
				_, err := svc.Refresh(ctx, blockedApp, "rt")
				return err
			},
			wantErr: auth.ErrEmailDomainBlocked,
		},
		{
			name: "refresh unknown token",
			run: func(svc *auth.Service, d authDeps) error {
				d.tokens.EXPECT().LookupRefresh(gomock.Any(), "bad").Return(token.RefreshRecord{}, token.ErrNotFound)
				_, err := svc.Refresh(ctx, application, "bad")
				return err
			},
			wantErr: auth.ErrInvalidCredentials,
		},
		{
			name: "logout revokes token",
			run: func(svc *auth.Service, d authDeps) error {
				d.tokens.EXPECT().Revoke(gomock.Any(), "rt").Return(nil)
				return svc.Logout(ctx, "rt")
			},
		},
		{
			name: "logout empty token",
			run: func(svc *auth.Service, _ authDeps) error {
				return svc.Logout(ctx, "")
			},
			wantErr: auth.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, deps := newAuthService(t)
			err := tt.run(svc, deps)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestServicePasswordEmailVerification(t *testing.T) {
	ctx := context.Background()
	appID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	email := "user@example.com"
	password := "ValidPass1"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	application := testApplication(appID, app.AuthMethodPassword, true)
	application.Settings.EmailVerificationRequired = true
	unverified := users.User{ID: userID, ApplicationID: appID, Email: email, Status: users.StatusActive}

	t.Run("signup does not mark verified before otp", func(t *testing.T) {
		svc, deps := newAuthService(t)
		deps.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(users.User{}, users.ErrNotFound)
		deps.users.EXPECT().CreateWithCredential(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, user users.User, _ string) (users.User, error) {
				if user.EmailVerifiedAt != nil {
					t.Fatal("must not set email_verified_at on create")
				}
				return unverified, nil
			},
		)
		deps.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(unverified, nil)
		deps.challenges.EXPECT().IssueOTP(gomock.Any(), appID, gomock.Any(), email, 6, 300).Return("123456", 5*time.Minute, nil)
		deps.mailer.EXPECT().SendOTP(gomock.Any(), gomock.Any(), email, "123456").Return(nil)

		got, err := svc.Authenticate(ctx, application, auth.Request{Email: email, Password: password, IP: "127.0.0.1"})
		if err != nil {
			t.Fatalf("Authenticate: %v", err)
		}
		if got.Status != auth.StatusChallengeSent {
			t.Fatalf("status = %s", got.Status)
		}
	})

	t.Run("otp success marks verified", func(t *testing.T) {
		svc, deps := newAuthService(t)
		pair := token.Pair{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 900, TokenType: "Bearer"}
		deps.users.EXPECT().GetByEmail(gomock.Any(), appID, email).Return(unverified, nil)
		deps.users.EXPECT().GetPasswordHash(gomock.Any(), userID).Return(hash, nil)
		deps.challenges.EXPECT().VerifyOTP(gomock.Any(), appID, email, "123456", 5).Return(challenge.Challenge{Target: email}, nil)
		deps.users.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, user users.User) (users.User, error) {
				if user.EmailVerifiedAt == nil {
					t.Fatal("must set email_verified_at after otp")
				}
				return user, nil
			},
		)
		deps.tokens.EXPECT().Issue(gomock.Any(), gomock.Any(), gomock.Any()).Return(pair, nil)

		got, err := svc.Authenticate(ctx, application, auth.Request{Email: email, Password: password, Code: "123456"})
		if err != nil {
			t.Fatalf("Authenticate: %v", err)
		}
		if got.Status != auth.StatusAuthenticated {
			t.Fatalf("status = %s", got.Status)
		}
	})
}
