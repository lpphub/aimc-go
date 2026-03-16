package jwt

import (
	"fmt"
	"sync"

	"github.com/lpphub/goweb/pkg/jwt"
	"aimc-go/infra"
)

type (
	Claims    = jwt.Claims
	TokenPair = jwt.TokenPair
)

var (
	jm   *jwt.Manager
	once sync.Once
)

func useJwt() *jwt.Manager {
	once.Do(func() {
		jm, _ = jwt.NewManager(jwt.Config{
			Secret:           infra.Cfg.JWT.Secret,
			AccessExpireSec:  infra.Cfg.JWT.ExpireTime,
			RefreshExpireSec: infra.Cfg.JWT.RefreshExpireTime,
		})
	})
	return jm
}

func ParseAccessToken(tokenString string) (*Claims, error) {
	claims, err := useJwt().ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != jwt.AccessTokenType {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}

func GenerateTokenPair(userID uint) (*TokenPair, error) {
	token, err := useJwt().GenerateTokenPair(userID)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func RefreshToken(refreshToken string) (*TokenPair, error) {
	token, err := useJwt().RefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}
	return token, nil
}