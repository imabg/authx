package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SMTPRepository struct {
	pool *pgxpool.Pool
}

func NewSMTPRepository(pool *pgxpool.Pool) ISMTPRepository {
	return &SMTPRepository{pool: pool}
}

const smtpSelect = `
	SELECT id, application_id, name, host, port, username_ciphertext, password_ciphertext,
	       tls, skip_verify, active, COALESCE(updated_by, ''), creation_timestamp, updated_timestamp
	FROM application_smtp_configs
`

func (r *SMTPRepository) Create(ctx context.Context, cfg SMTPConfig) (SMTPConfig, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO application_smtp_configs (
			application_id, name, host, port, username_ciphertext, password_ciphertext,
			tls, skip_verify, active, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, FALSE, $9)
		RETURNING id, application_id, name, host, port, username_ciphertext, password_ciphertext,
		          tls, skip_verify, active, COALESCE(updated_by, ''), creation_timestamp, updated_timestamp
	`, cfg.ApplicationID, cfg.Name, cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.TLS, cfg.SkipVerify, nullIfEmpty(cfg.UpdatedBy))
	return scanSMTPConfig(row)
}

func (r *SMTPRepository) ListByApplication(ctx context.Context, applicationID uuid.UUID) ([]SMTPConfig, error) {
	rows, err := r.pool.Query(ctx, smtpSelect+`
		WHERE application_id = $1
		ORDER BY creation_timestamp ASC
	`, applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SMTPConfig
	for rows.Next() {
		cfg, err := scanSMTPConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cfg)
	}
	return out, rows.Err()
}

func (r *SMTPRepository) GetByID(ctx context.Context, applicationID, id uuid.UUID) (SMTPConfig, error) {
	row := r.pool.QueryRow(ctx, smtpSelect+`
		WHERE application_id = $1 AND id = $2
	`, applicationID, id)
	cfg, err := scanSMTPConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SMTPConfig{}, ErrSMTPNotFound
		}
		return SMTPConfig{}, err
	}
	return cfg, nil
}

func (r *SMTPRepository) GetActive(ctx context.Context, applicationID uuid.UUID) (SMTPConfig, error) {
	row := r.pool.QueryRow(ctx, smtpSelect+`
		WHERE application_id = $1 AND active
	`, applicationID)
	cfg, err := scanSMTPConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SMTPConfig{}, ErrSMTPNotFound
		}
		return SMTPConfig{}, err
	}
	return cfg, nil
}

func (r *SMTPRepository) Update(ctx context.Context, cfg SMTPConfig) (SMTPConfig, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE application_smtp_configs
		SET name = $3,
		    host = $4,
		    port = $5,
		    username_ciphertext = $6,
		    password_ciphertext = $7,
		    tls = $8,
		    skip_verify = $9,
		    updated_by = $10,
		    updated_timestamp = CURRENT_TIMESTAMP
		WHERE application_id = $1 AND id = $2
		RETURNING id, application_id, name, host, port, username_ciphertext, password_ciphertext,
		          tls, skip_verify, active, COALESCE(updated_by, ''), creation_timestamp, updated_timestamp
	`, cfg.ApplicationID, cfg.ID, cfg.Name, cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.TLS, cfg.SkipVerify, nullIfEmpty(cfg.UpdatedBy))
	updated, err := scanSMTPConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SMTPConfig{}, ErrSMTPNotFound
		}
		return SMTPConfig{}, err
	}
	return updated, nil
}

func (r *SMTPRepository) Activate(ctx context.Context, applicationID, id uuid.UUID) (SMTPConfig, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SMTPConfig{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE application_smtp_configs
		SET active = FALSE, updated_timestamp = CURRENT_TIMESTAMP
		WHERE application_id = $1 AND active
	`, applicationID); err != nil {
		return SMTPConfig{}, err
	}

	row := tx.QueryRow(ctx, `
		UPDATE application_smtp_configs
		SET active = TRUE, updated_timestamp = CURRENT_TIMESTAMP
		WHERE application_id = $1 AND id = $2
		RETURNING id, application_id, name, host, port, username_ciphertext, password_ciphertext,
		          tls, skip_verify, active, COALESCE(updated_by, ''), creation_timestamp, updated_timestamp
	`, applicationID, id)
	cfg, err := scanSMTPConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SMTPConfig{}, ErrSMTPNotFound
		}
		return SMTPConfig{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SMTPConfig{}, fmt.Errorf("activate smtp config: %w", err)
	}
	return cfg, nil
}

func (r *SMTPRepository) Delete(ctx context.Context, applicationID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM application_smtp_configs
		WHERE application_id = $1 AND id = $2
	`, applicationID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSMTPNotFound
	}
	return nil
}

func scanSMTPConfig(row scanner) (SMTPConfig, error) {
	var cfg SMTPConfig
	if err := row.Scan(
		&cfg.ID,
		&cfg.ApplicationID,
		&cfg.Name,
		&cfg.Host,
		&cfg.Port,
		&cfg.Username,
		&cfg.Password,
		&cfg.TLS,
		&cfg.SkipVerify,
		&cfg.Active,
		&cfg.UpdatedBy,
		&cfg.CreationTimestamp,
		&cfg.UpdatedTimestamp,
	); err != nil {
		return SMTPConfig{}, err
	}
	return cfg, nil
}
