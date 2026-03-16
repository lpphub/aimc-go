// contract/user.go
package contract

import "context"

type UserBiz interface {
	Create(ctx context.Context, email, password string) (*UserDTO, error)
	Get(ctx context.Context, userID uint) (*UserDTO, error)
	ValidateLogin(ctx context.Context, email, password string) (*UserDTO, error)
}

type UserDTO struct {
	ID       uint
	Email    string
	Name     string
	Password string
}