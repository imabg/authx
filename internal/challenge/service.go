package challenge

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo IRepository
}

func NewService(repo IRepository) *Service {
	return &Service{repo: repo}
}

var _ IService = (*Service)(nil)

func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func secretsEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *Service) IssueOTP(ctx context.Context, applicationID uuid.UUID, userID *uuid.UUID, email string, length, ttlSeconds int) (string, time.Duration, error) {
	code, err := randomDigits(length)
	if err != nil {
		return "", 0, err
	}
	ttl := time.Duration(ttlSeconds) * time.Second
	if err := s.repo.InvalidateOpen(ctx, applicationID, email, TypeOTP); err != nil {
		return "", 0, err
	}
	_, err = s.repo.Create(ctx, Challenge{
		ApplicationID: applicationID,
		UserID:        userID,
		Type:          TypeOTP,
		Target:        email,
		SecretHash:    HashSecret(code),
		ExpiresAt:     time.Now().Add(ttl),
	})
	if err != nil {
		return "", 0, err
	}
	return code, ttl, nil
}

func (s *Service) VerifyOTP(ctx context.Context, applicationID uuid.UUID, email, code string, maxAttempts int) (Challenge, error) {
	c, err := s.repo.GetOpenByTarget(ctx, applicationID, email, TypeOTP)
	if err != nil {
		return Challenge{}, err
	}
	attempts, err := s.repo.IncrementAttempts(ctx, c.ID)
	if err != nil {
		return Challenge{}, err
	}
	if attempts > maxAttempts {
		_ = s.repo.Consume(ctx, c.ID)
		return Challenge{}, ErrNotFound
	}
	if !secretsEqual(c.SecretHash, HashSecret(code)) {
		return Challenge{}, ErrNotFound
	}
	if err := s.repo.Consume(ctx, c.ID); err != nil {
		return Challenge{}, err
	}
	return c, nil
}

func (s *Service) IssueMagicLink(ctx context.Context, applicationID uuid.UUID, userID *uuid.UUID, email string, ttlSeconds int) (string, time.Duration, error) {
	token, err := randomHex(32)
	if err != nil {
		return "", 0, err
	}
	ttl := time.Duration(ttlSeconds) * time.Second
	if err := s.repo.InvalidateOpen(ctx, applicationID, email, TypeMagicLink); err != nil {
		return "", 0, err
	}
	_, err = s.repo.Create(ctx, Challenge{
		ApplicationID: applicationID,
		UserID:        userID,
		Type:          TypeMagicLink,
		Target:        email,
		SecretHash:    HashSecret(token),
		ExpiresAt:     time.Now().Add(ttl),
	})
	if err != nil {
		return "", 0, err
	}
	return token, ttl, nil
}

func (s *Service) ConsumeMagicLink(ctx context.Context, applicationID uuid.UUID, token, email string) (Challenge, error) {
	c, err := s.repo.GetOpenByHash(ctx, applicationID, TypeMagicLink, HashSecret(token))
	if err != nil {
		return Challenge{}, err
	}
	if email != "" && c.Target != email {
		return Challenge{}, ErrNotFound
	}
	if err := s.repo.Consume(ctx, c.ID); err != nil {
		return Challenge{}, err
	}
	return c, nil
}

func randomDigits(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("invalid otp length")
	}
	max := big.NewInt(10)
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		d, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + d.Int64())
	}
	return string(out), nil
}

func randomHex(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
