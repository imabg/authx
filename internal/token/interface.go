package token

import (
	"context"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/users"
)

//go:generate mockgen -destination=mock/token_mock.go -package=mock -mock_names=IService=MockTokenService github.com/imabg/authx/internal/token IService

type IService interface {
	Issue(ctx context.Context, user users.User, application *app.Application) (Pair, error)
	ParseAccess(token string) (*Claims, error)
	Rotate(ctx context.Context, refreshToken string, applicationID uuid.UUID, user users.User, application *app.Application) (Pair, error)
	LookupRefresh(ctx context.Context, refreshToken string) (RefreshRecord, error)
	Revoke(ctx context.Context, refreshToken string) error
}
