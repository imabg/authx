package users_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/users"
	"github.com/imabg/authx/internal/users/mock"
	"go.uber.org/mock/gomock"
)

func TestServiceUpdateProfile(t *testing.T) {
	ctx := context.Background()
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	existing := users.User{ID: id, FirstName: "Ada", LastName: "Lovelace", Status: users.StatusActive}
	first := "Grace"

	ctrl := gomock.NewController(t)
	repo := mock.NewMockIUserRepository(ctrl)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(existing, nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, user users.User) (users.User, error) {
			if user.FirstName != "Grace" || user.LastName != "Lovelace" {
				t.Fatalf("updated = %+v", user)
			}
			return user, nil
		},
	)

	svc := users.NewService(repo)
	got, err := svc.UpdateProfile(ctx, id, users.ProfileUpdate{FirstName: &first})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if got.FirstName != "Grace" {
		t.Fatalf("first_name = %s", got.FirstName)
	}
}

func TestServiceUpdateProfileNotFound(t *testing.T) {
	ctx := context.Background()
	id := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	first := "Grace"

	ctrl := gomock.NewController(t)
	repo := mock.NewMockIUserRepository(ctrl)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(users.User{}, users.ErrNotFound)

	svc := users.NewService(repo)
	_, err := svc.UpdateProfile(ctx, id, users.ProfileUpdate{FirstName: &first})
	if !errors.Is(err, users.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestServiceUpdateProfileValidation(t *testing.T) {
	ctx := context.Background()
	id := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	existing := users.User{ID: id, FirstName: "Ada"}
	tooLong := strings.Repeat("a", users.MaxFirstNameLen+1)

	ctrl := gomock.NewController(t)
	repo := mock.NewMockIUserRepository(ctrl)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(existing, nil)

	svc := users.NewService(repo)
	_, err := svc.UpdateProfile(ctx, id, users.ProfileUpdate{FirstName: &tooLong})
	if !errors.Is(err, users.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestServiceGetByID(t *testing.T) {
	ctx := context.Background()
	id := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	existing := users.User{ID: id, Email: "ada@example.com"}

	ctrl := gomock.NewController(t)
	repo := mock.NewMockIUserRepository(ctrl)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(existing, nil)

	svc := users.NewService(repo)
	got, err := svc.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Email != existing.Email {
		t.Fatalf("got = %+v", got)
	}
}
