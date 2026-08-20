package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("refresh token not found")

type RefreshRecord struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ApplicationID uuid.UUID
	ExpiresAt     time.Time
	RevokedAt     *time.Time
}

type RefreshStore struct {
	pool *pgxpool.Pool
}

func NewRefreshStore(pool *pgxpool.Pool) *RefreshStore {
	return &RefreshStore{pool: pool}
}

func HashRefresh(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func NewOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *RefreshStore) Create(ctx context.Context, userID, applicationID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, application_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, userID, applicationID, tokenHash, expiresAt)
	return err
}

func (s *RefreshStore) GetActive(ctx context.Context, tokenHash string) (RefreshRecord, error) {
	var rec RefreshRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, application_id, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(&rec.ID, &rec.UserID, &rec.ApplicationID, &rec.ExpiresAt, &rec.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshRecord{}, ErrNotFound
		}
		return RefreshRecord{}, err
	}
	if rec.RevokedAt != nil || time.Now().After(rec.ExpiresAt) {
		return RefreshRecord{}, ErrNotFound
	}
	return rec, nil
}

func (s *RefreshStore) Revoke(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

func (s *RefreshStore) RevokeID(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND revoked_at IS NULL
	`, id)
	return err
}
