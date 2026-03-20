// modules/auth/init.go
package auth

import (
	"aimc-go/app/modules/core"
	"aimc-go/app/shared/contracts"

	"github.com/gin-gonic/gin"
)

var _ core.Module = (*Module)(nil)

type Module struct {
	Service *Service
	handler *Handler
}

func New(userSvc contracts.UserBiz) *Module {
	svc := NewService(userSvc)
	h := NewHandler(svc)

	return &Module{
		Service: svc,
		handler: h,
	}
}

func (m *Module) Routes(r *gin.RouterGroup) {
	m.handler.register(r)
}
