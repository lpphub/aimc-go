// module/auth/service.go
package auth

import (
	"context"

	"github.com/jinzhu/copier"
	"aimc-go/contract"
	"aimc-go/infra/jwt"
)

type Service struct {
	userSvc contract.UserBiz
}

func NewService(userSvc contract.UserBiz) *Service {
	return &Service{userSvc: userSvc}
}

func (s *Service) Register(ctx context.Context, email, password string) (*AuthResp, error) {
	newUser, err := s.userSvc.Create(ctx, email, password)
	if err != nil {
		return nil, err
	}

	tokenPair, err := jwt.GenerateTokenPair(newUser.ID)
	if err != nil {
		return nil, err
	}

	var authUser AuthUser
	_ = copier.Copy(&authUser, newUser)

	return &AuthResp{
		User:         &authUser,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*AuthResp, error) {
	loginUser, err := s.userSvc.ValidateLogin(ctx, email, password)
	if err != nil {
		return nil, err
	}

	tokenPair, err := jwt.GenerateTokenPair(loginUser.ID)
	if err != nil {
		return nil, err
	}

	var authUser AuthUser
	_ = copier.Copy(&authUser, loginUser)

	return &AuthResp{
		User:         &authUser,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

func (s *Service) RefreshToken(_ context.Context, refreshToken string) (*AuthResp, error) {
	tokenPair, err := jwt.RefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return &AuthResp{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}