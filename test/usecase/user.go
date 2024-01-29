package usecase

import (
	"context"

	"github.com/lkeix/go-test-generator/test/domain/model"
	"github.com/lkeix/go-test-generator/test/domain/repository"
)

type User interface {
	Create(ctx context.Context, firstName, lastName string) error
}

type user struct {
	userRopo repository.User
}

func NewUser(userRepo repository.User) User {
	return &user{
		userRopo: userRepo,
	}
}

func (u *user) Create(ctx context.Context, firstName, lastName string) error {
	user := model.NewUser(firstName, lastName)
	if err := u.userRopo.Create(ctx, user); err != nil {
		return err
	}

	return nil
}
