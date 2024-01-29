package infra

import (
	"context"
	"sync"

	"github.com/lkeix/go-test-generator/test/domain/model"
	"github.com/lkeix/go-test-generator/test/domain/repository"
)

type user struct {
	mux   sync.Mutex
	store map[string]*model.User
}

func NewUser() repository.User {
	return &user{
		mux:   sync.Mutex{},
		store: make(map[string]*model.User),
	}
}

func (u *user) Create(ctx context.Context, user *model.User) error {
	// skelton
	return nil
}
