package challenge

import (
	"context"
	"time"

	"github.com/google/uuid"
)

//go:generate mockgen -destination=mock/challenge_mock.go -package=mock -mock_names=IRepository=MockChallengeRepository,IService=MockChallengeService github.com/imabg/authx/internal/challenge IRepository,IService

type IRepository interface {
	Create(ctx context.Context, c Challenge) (Challenge, error)
	InvalidateOpen(ctx context.Context, applicationID uuid.UUID, target, challengeType string) error
	GetOpenByTarget(ctx context.Context, applicationID uuid.UUID, target, challengeType string) (Challenge, error)
	GetOpenByHash(ctx context.Context, applicationID uuid.UUID, challengeType, secretHash string) (Challenge, error)
	IncrementAttempts(ctx context.Context, id uuid.UUID) (int, error)
	Consume(ctx context.Context, id uuid.UUID) error
}

type IService interface {
	IssueOTP(ctx context.Context, applicationID uuid.UUID, userID *uuid.UUID, email string, length, ttlSeconds int) (string, time.Duration, error)
	VerifyOTP(ctx context.Context, applicationID uuid.UUID, email, code string, maxAttempts int) (Challenge, error)
	IssueMagicLink(ctx context.Context, applicationID uuid.UUID, userID *uuid.UUID, email string, ttlSeconds int) (string, time.Duration, error)
	ConsumeMagicLink(ctx context.Context, applicationID uuid.UUID, token, email string) (Challenge, error)
}
