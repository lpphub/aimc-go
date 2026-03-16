package errs

import (
	"net/http"

	"github.com/lpphub/goweb/base"
)

var (
	ErrServerError = base.NewErrorWithStatus(500, "server internal error", http.StatusInternalServerError)

	ErrNoToken      = base.NewErrorWithStatus(1000, "no token", http.StatusUnauthorized)
	ErrInvalidToken = base.NewErrorWithStatus(1001, "invalid token", http.StatusUnauthorized)
	ErrInvalidParam = base.NewError(1100, "invalid param")

	ErrUserExists      = base.NewError(2101, "user already exists")
	ErrUserNotFound    = base.NewError(2102, "user not found")
	ErrInvalidPassword = base.NewError(2103, "invalid password")
	ErrLoginFailed     = base.NewError(2104, "login failed")
	ErrUserDisabled    = base.NewError(2105, "user is disabled")

	ErrPostNotFound = base.NewError(2201, "post not found")
)