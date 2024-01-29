// go:generate mockgen -source=./domain/repository/user.go -destination=./mock/repository/user.go
package repository

import (
	"context"

	"github.com/lkeix/go-test-generator/test/domain/model"
)

type User interface {
	Create(ctx context.Context, user *model.User) error
}
