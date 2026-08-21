package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) IUserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user User) (User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (application_id, email, first_name, last_name, signup_method, email_verified_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, application_id, email, first_name, last_name, COALESCE(signup_method, ''), email_verified_at, status, creation_timestamp, updated_timestamp
	`, user.ApplicationID, user.Email, user.FirstName, user.LastName, nullIfEmpty(user.SignupMethod), user.EmailVerifiedAt, statusOrActive(user.Status))
	return scanUser(row)
}

func (r *UserRepository) CreateWithCredential(ctx context.Context, user User, passwordHash string) (User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO users (application_id, email, first_name, last_name, signup_method, email_verified_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, application_id, email, first_name, last_name, COALESCE(signup_method, ''), email_verified_at, status, creation_timestamp, updated_timestamp
	`, user.ApplicationID, user.Email, user.FirstName, user.LastName, nullIfEmpty(user.SignupMethod), user.EmailVerifiedAt, statusOrActive(user.Status))
	created, err := scanUser(row)
	if err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO credentials (user_id, password_hash)
		VALUES ($1, $2)
	`, created.ID, passwordHash); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return created, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, application_id, email, first_name, last_name, COALESCE(signup_method, ''), email_verified_at, status, creation_timestamp, updated_timestamp
		FROM users
		WHERE id = $1
	`, id)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, applicationID uuid.UUID, email string) (User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, application_id, email, first_name, last_name, COALESCE(signup_method, ''), email_verified_at, status, creation_timestamp, updated_timestamp
		FROM users
		WHERE application_id = $1 AND email = $2
	`, applicationID, email)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user User) (User, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE users
		SET first_name = $2,
		    last_name = $3,
		    signup_method = $4,
		    email_verified_at = $5,
		    status = $6,
		    updated_timestamp = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id, application_id, email, first_name, last_name, COALESCE(signup_method, ''), email_verified_at, status, creation_timestamp, updated_timestamp
	`, user.ID, user.FirstName, user.LastName, nullIfEmpty(user.SignupMethod), user.EmailVerifiedAt, statusOrActive(user.Status))
	updated, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return updated, nil
}

func (r *UserRepository) GetPasswordHash(ctx context.Context, userID uuid.UUID) (string, error) {
	var hash string
	err := r.pool.QueryRow(ctx, `SELECT password_hash FROM credentials WHERE user_id = $1`, userID).Scan(&hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return hash, nil
}

func (r *UserRepository) UpsertCredential(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO credentials (user_id, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
		SET password_hash = EXCLUDED.password_hash, updated_timestamp = CURRENT_TIMESTAMP
	`, userID, passwordHash)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (User, error) {
	var user User
	if err := row.Scan(
		&user.ID,
		&user.ApplicationID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.SignupMethod,
		&user.EmailVerifiedAt,
		&user.Status,
		&user.CreationTimestamp,
		&user.UpdatedTimestamp,
	); err != nil {
		return User{}, err
	}
	user.MarkVerified()
	return user, nil
}

func statusOrActive(status string) string {
	if status == "" {
		return StatusActive
	}
	return status
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}
