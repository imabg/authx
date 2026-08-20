package challenge

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("challenge not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var _ IRepository = (*Repository)(nil)

func (r *Repository) Create(ctx context.Context, c Challenge) (Challenge, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO auth_challenges (application_id, user_id, type, target, secret_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, application_id, user_id, type, target, secret_hash, expires_at, consumed_at, attempt_count, creation_timestamp
	`, c.ApplicationID, c.UserID, c.Type, c.Target, c.SecretHash, c.ExpiresAt)
	return scanChallenge(row)
}

func (r *Repository) InvalidateOpen(ctx context.Context, applicationID uuid.UUID, target, challengeType string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE auth_challenges
		SET consumed_at = CURRENT_TIMESTAMP
		WHERE application_id = $1
		  AND target = $2
		  AND type = $3
		  AND consumed_at IS NULL
	`, applicationID, target, challengeType)
	return err
}

func (r *Repository) GetOpenByTarget(ctx context.Context, applicationID uuid.UUID, target, challengeType string) (Challenge, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, application_id, user_id, type, target, secret_hash, expires_at, consumed_at, attempt_count, creation_timestamp
		FROM auth_challenges
		WHERE application_id = $1
		  AND target = $2
		  AND type = $3
		  AND consumed_at IS NULL
		  AND expires_at > CURRENT_TIMESTAMP
		ORDER BY creation_timestamp DESC
		LIMIT 1
	`, applicationID, target, challengeType)
	c, err := scanChallenge(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Challenge{}, ErrNotFound
		}
		return Challenge{}, err
	}
	return c, nil
}

func (r *Repository) GetOpenByHash(ctx context.Context, applicationID uuid.UUID, challengeType, secretHash string) (Challenge, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, application_id, user_id, type, target, secret_hash, expires_at, consumed_at, attempt_count, creation_timestamp
		FROM auth_challenges
		WHERE application_id = $1
		  AND type = $2
		  AND secret_hash = $3
		  AND consumed_at IS NULL
		  AND expires_at > CURRENT_TIMESTAMP
		ORDER BY creation_timestamp DESC
		LIMIT 1
	`, applicationID, challengeType, secretHash)
	c, err := scanChallenge(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Challenge{}, ErrNotFound
		}
		return Challenge{}, err
	}
	return c, nil
}

func (r *Repository) IncrementAttempts(ctx context.Context, id uuid.UUID) (int, error) {
	var attempts int
	err := r.pool.QueryRow(ctx, `
		UPDATE auth_challenges
		SET attempt_count = attempt_count + 1
		WHERE id = $1
		RETURNING attempt_count
	`, id).Scan(&attempts)
	return attempts, err
}

func (r *Repository) Consume(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE auth_challenges
		SET consumed_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND consumed_at IS NULL
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanChallenge(row scanner) (Challenge, error) {
	var c Challenge
	if err := row.Scan(
		&c.ID,
		&c.ApplicationID,
		&c.UserID,
		&c.Type,
		&c.Target,
		&c.SecretHash,
		&c.ExpiresAt,
		&c.ConsumedAt,
		&c.AttemptCount,
		&c.CreatedAt,
	); err != nil {
		return Challenge{}, err
	}
	return c, nil
}
