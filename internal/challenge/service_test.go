package challenge_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/challenge"
	"github.com/imabg/authx/internal/challenge/mock"
	"go.uber.org/mock/gomock"
)

func TestServiceVerifyOTP(t *testing.T) {
	ctx := context.Background()
	appID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	chID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	email := "user@example.com"
	code := "123456"
	open := challenge.Challenge{
		ID:            chID,
		ApplicationID: appID,
		Type:          challenge.TypeOTP,
		Target:        email,
		SecretHash:    challenge.HashSecret(code),
	}

	tests := []struct {
		name        string
		code        string
		maxAttempts int
		setup       func(*mock.MockChallengeRepository)
		wantErr     error
	}{
		{
			name:        "valid code",
			code:        code,
			maxAttempts: 5,
			setup: func(repo *mock.MockChallengeRepository) {
				repo.EXPECT().GetOpenByTarget(gomock.Any(), appID, email, challenge.TypeOTP).Return(open, nil)
				repo.EXPECT().IncrementAttempts(gomock.Any(), chID).Return(1, nil)
				repo.EXPECT().Consume(gomock.Any(), chID).Return(nil)
			},
		},
		{
			name:        "wrong code",
			code:        "000000",
			maxAttempts: 5,
			setup: func(repo *mock.MockChallengeRepository) {
				repo.EXPECT().GetOpenByTarget(gomock.Any(), appID, email, challenge.TypeOTP).Return(open, nil)
				repo.EXPECT().IncrementAttempts(gomock.Any(), chID).Return(1, nil)
			},
			wantErr: challenge.ErrNotFound,
		},
		{
			name:        "max attempts exceeded",
			code:        code,
			maxAttempts: 5,
			setup: func(repo *mock.MockChallengeRepository) {
				repo.EXPECT().GetOpenByTarget(gomock.Any(), appID, email, challenge.TypeOTP).Return(open, nil)
				repo.EXPECT().IncrementAttempts(gomock.Any(), chID).Return(6, nil)
				repo.EXPECT().Consume(gomock.Any(), chID).Return(nil)
			},
			wantErr: challenge.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock.NewMockChallengeRepository(ctrl)
			tt.setup(repo)
			svc := challenge.NewService(repo)
			got, err := svc.VerifyOTP(ctx, appID, email, tt.code, tt.maxAttempts)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("VerifyOTP: %v", err)
			}
			if got.ID != chID {
				t.Fatalf("id = %s, want %s", got.ID, chID)
			}
		})
	}
}

func TestServiceConsumeMagicLink(t *testing.T) {
	ctx := context.Background()
	appID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	chID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	tokenValue := "magic-token"
	open := challenge.Challenge{
		ID:            chID,
		ApplicationID: appID,
		Type:          challenge.TypeMagicLink,
		Target:        "user@example.com",
		SecretHash:    challenge.HashSecret(tokenValue),
	}

	tests := []struct {
		name    string
		email   string
		setup   func(*mock.MockChallengeRepository)
		wantErr error
	}{
		{
			name:  "consumes matching token",
			email: "user@example.com",
			setup: func(repo *mock.MockChallengeRepository) {
				repo.EXPECT().GetOpenByHash(gomock.Any(), appID, challenge.TypeMagicLink, challenge.HashSecret(tokenValue)).Return(open, nil)
				repo.EXPECT().Consume(gomock.Any(), chID).Return(nil)
			},
		},
		{
			name:  "rejects email mismatch",
			email: "other@example.com",
			setup: func(repo *mock.MockChallengeRepository) {
				repo.EXPECT().GetOpenByHash(gomock.Any(), appID, challenge.TypeMagicLink, challenge.HashSecret(tokenValue)).Return(open, nil)
			},
			wantErr: challenge.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock.NewMockChallengeRepository(ctrl)
			tt.setup(repo)
			svc := challenge.NewService(repo)
			got, err := svc.ConsumeMagicLink(ctx, appID, tokenValue, tt.email)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ConsumeMagicLink: %v", err)
			}
			if got.ID != chID {
				t.Fatalf("id = %s, want %s", got.ID, chID)
			}
		})
	}
}
