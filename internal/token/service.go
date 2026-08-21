package token

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/users"
	"github.com/imabg/authx/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	TokenType    string
}

type Service struct {
	jwt     *JWTSigner
	refresh *RefreshStore
}

func NewService(cfg config.ApplicationConfig, pool *pgxpool.Pool) IService {
	return &Service{
		jwt:     NewJWTSigner(cfg),
		refresh: NewRefreshStore(pool),
	}
}

func (s *Service) Issue(ctx context.Context, user users.User, application *app.Application) (Pair, error) {
	accessTTL := time.Duration(application.Settings.Tokens.AccessTTLSeconds) * time.Second
	refreshTTL := time.Duration(application.Settings.Tokens.RefreshTTLSeconds) * time.Second
	access, err := s.jwt.Sign(user.ID.String(), application.ID.String(), user.Email, accessTTL)
	if err != nil {
		return Pair{}, err
	}
	refresh, err := NewOpaqueToken()
	if err != nil {
		return Pair{}, err
	}
	if err := s.refresh.Create(ctx, user.ID, application.ID, HashRefresh(refresh), time.Now().Add(refreshTTL)); err != nil {
		return Pair{}, err
	}
	return Pair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(accessTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *Service) ParseAccess(token string) (*Claims, error) {
	return s.jwt.Parse(token)
}

func (s *Service) Rotate(ctx context.Context, refreshToken string, applicationID uuid.UUID, user users.User, application *app.Application) (Pair, error) {
	rec, err := s.refresh.GetActive(ctx, HashRefresh(refreshToken))
	if err != nil {
		return Pair{}, err
	}
	if rec.ApplicationID != applicationID || rec.UserID != user.ID {
		return Pair{}, ErrNotFound
	}
	if err := s.refresh.RevokeID(ctx, rec.ID); err != nil {
		return Pair{}, err
	}
	return s.Issue(ctx, user, application)
}

func (s *Service) LookupRefresh(ctx context.Context, refreshToken string) (RefreshRecord, error) {
	return s.refresh.GetActive(ctx, HashRefresh(refreshToken))
}

func (s *Service) Revoke(ctx context.Context, refreshToken string) error {
	return s.refresh.Revoke(ctx, HashRefresh(refreshToken))
}
