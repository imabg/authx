package app

import (
	"context"

	"github.com/google/uuid"
)

//go:generate mockgen -destination=mock/app_mock.go -package=mock -mock_names=IRepository=MockAppRepository github.com/imabg/authx/internal/app IRepository

type IRepository interface {
	Create(ctx context.Context, application Application) (Application, error)
	GetByID(ctx context.Context, id uuid.UUID) (Application, error)
	GetByClientID(ctx context.Context, clientID string) (Application, error)
	ExistsByClientID(ctx context.Context, clientID string) (bool, error)
	Update(ctx context.Context, application Application) (Application, error)
}
