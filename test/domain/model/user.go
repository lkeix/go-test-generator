package model

import "github.com/google/uuid"

type User struct {
	ID        string
	FirstName string
	LastName  string
}

func NewUser(firstName, lastName string) *User {
	id := uuid.New()
	return &User{
		ID:        id.String(),
		FirstName: firstName,
		LastName:  lastName,
	}
}
