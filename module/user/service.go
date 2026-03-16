// module/user/service.go
package user

import (
	"context"
	"errors"
	"time"

	"aimc-go/contract"
	"aimc-go/shared/errs"
	"aimc-go/shared/strutils"

	"gorm.io/gorm"
)

var _ contract.UserBiz = (*Service)(nil)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, email, password string) (*contract.UserDTO, error) {
	exists, _ := s.repo.ExistsByEmail(ctx, email)
	if exists {
		return nil, errs.ErrUserExists
	}

	user := &User{
		Name:      strutils.ExtractNameFromEmail(email),
		Email:     email,
		Status:    StatusActive,
		CreatedAt: time.Now(),
	}

	if err := user.SetPassword(password); err != nil {
		return nil, errs.ErrInvalidPassword
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return toDTO(user), nil
}

func (s *Service) Get(ctx context.Context, userID uint) (*contract.UserDTO, error) {
	user, err := s.repo.First(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toDTO(user), nil
}

func (s *Service) ValidateLogin(ctx context.Context, email, password string) (*contract.UserDTO, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrLoginFailed
		}
		return nil, err
	}

	if !user.IsActive() {
		return nil, errs.ErrUserDisabled
	}

	if err := user.ValidatePassword(password); err != nil {
		return nil, errs.ErrLoginFailed
	}

	return toDTO(user), nil
}

func toDTO(user *User) *contract.UserDTO {
	return &contract.UserDTO{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Password: user.Password,
	}
}
