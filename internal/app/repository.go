package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) IRepository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, application Application) (Application, error) {
	settingsJSON, err := json.Marshal(application.Settings)
	if err != nil {
		return Application{}, err
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO applications (name, description, client_id, client_secret_hash, settings, status, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, description, client_id, client_secret_hash, settings, status, COALESCE(updated_by, ''), creation_timestamp, updated_timestamp
	`, application.Name, nullIfEmpty(application.Description), application.ClientID, application.ClientSecretHash, settingsJSON, application.Status, nullIfEmpty(application.UpdatedBy))
	return scanApplication(row)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Application, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, description, client_id, client_secret_hash, settings, status, COALESCE(updated_by, ''), creation_timestamp, updated_timestamp
		FROM applications
		WHERE id = $1
	`, id)
	application, err := scanApplication(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Application{}, ErrNotFound
		}
		return Application{}, err
	}
	return application, nil
}

func (r *Repository) GetByClientID(ctx context.Context, clientID string) (Application, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, description, client_id, client_secret_hash, settings, status, COALESCE(updated_by, ''), creation_timestamp, updated_timestamp
		FROM applications
		WHERE client_id = $1
	`, clientID)
	application, err := scanApplication(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Application{}, ErrNotFound
		}
		return Application{}, err
	}
	return application, nil
}

func (r *Repository) ExistsByClientID(ctx context.Context, clientID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM applications WHERE client_id = $1)`, clientID).Scan(&exists)
	return exists, err
}

func (r *Repository) Update(ctx context.Context, application Application) (Application, error) {
	settingsJSON, err := json.Marshal(application.Settings)
	if err != nil {
		return Application{}, err
	}
	row := r.pool.QueryRow(ctx, `
		UPDATE applications
		SET name = $2,
		    description = $3,
		    settings = $4,
		    status = $5,
		    updated_by = $6,
		    updated_timestamp = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id, name, description, client_id, client_secret_hash, settings, status, COALESCE(updated_by, ''), creation_timestamp, updated_timestamp
	`, application.ID, application.Name, nullIfEmpty(application.Description), settingsJSON, application.Status, nullIfEmpty(application.UpdatedBy))
	updated, err := scanApplication(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Application{}, ErrNotFound
		}
		return Application{}, err
	}
	return updated, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanApplication(row scanner) (Application, error) {
	var (
		application Application
		description *string
		rawSettings []byte
	)
	if err := row.Scan(
		&application.ID,
		&application.Name,
		&description,
		&application.ClientID,
		&application.ClientSecretHash,
		&rawSettings,
		&application.Status,
		&application.UpdatedBy,
		&application.CreationTimestamp,
		&application.UpdatedTimestamp,
	); err != nil {
		return Application{}, err
	}
	if description != nil {
		application.Description = *description
	}
	settings, err := DecodeSettings(rawSettings)
	if err != nil {
		return Application{}, err
	}
	application.Settings = settings
	return application, nil
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

var ErrNotFound = fmt.Errorf("application not found")
