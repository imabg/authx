package users

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo IUserRepository
}

func NewService(repo IUserRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateProfile loads the user, applies a validated profile patch, and persists it.
func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, patch ProfileUpdate) (User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	patched, err := ApplyProfileUpdate(user, patch)
	if err != nil {
		return User{}, err
	}
	return s.repo.Update(ctx, patched)
}
