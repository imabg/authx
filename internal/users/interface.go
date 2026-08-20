package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

//go:generate mockgen -destination=mock/users_mock.go -package=mock github.com/imabg/authx/internal/users IUserRepository

var ErrNotFound = errors.New("user not found")

type IUserRepository interface {
	Create(ctx context.Context, user User) (User, error)
	CreateWithCredential(ctx context.Context, user User, passwordHash string) (User, error)
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	GetByEmail(ctx context.Context, applicationID uuid.UUID, email string) (User, error)
	Update(ctx context.Context, user User) (User, error)
	GetPasswordHash(ctx context.Context, userID uuid.UUID) (string, error)
	UpsertCredential(ctx context.Context, userID uuid.UUID, passwordHash string) error
}
