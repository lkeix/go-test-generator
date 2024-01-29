package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lkeix/go-test-generator/test/usecase"
)

type User struct {
	userUsecase usecase.User
}

type createUserHTTPRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (u *User) Create(c echo.Context) error {
	b := new(createUserHTTPRequest)
	if err := c.Bind(&b); err != nil {
		return err
	}

	ctx := c.Request().Context()
	if err := u.userUsecase.Create(ctx, b.FirstName, b.LastName); err != nil {
		return err
	}

	return c.NoContent(http.StatusCreated)
}
