package app

import (
	"context"

	"github.com/google/uuid"
)

// ISMTPRepository is the persistence adapter behind SMTPStore.
type ISMTPRepository interface {
	Create(ctx context.Context, cfg SMTPConfig) (SMTPConfig, error)
	ListByApplication(ctx context.Context, applicationID uuid.UUID) ([]SMTPConfig, error)
	GetByID(ctx context.Context, applicationID, id uuid.UUID) (SMTPConfig, error)
	GetActive(ctx context.Context, applicationID uuid.UUID) (SMTPConfig, error)
	Update(ctx context.Context, cfg SMTPConfig) (SMTPConfig, error)
	Activate(ctx context.Context, applicationID, id uuid.UUID) (SMTPConfig, error)
	Delete(ctx context.Context, applicationID, id uuid.UUID) error
}
