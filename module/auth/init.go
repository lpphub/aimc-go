// module/auth/init.go
package auth

import (
	"github.com/gin-gonic/gin"
	"aimc-go/contract"
	"aimc-go/server/core"
)

var _ core.Module = (*Module)(nil)

type Module struct {
	Service *Service
	handler *Handler
}

func New(userSvc contract.UserBiz) *Module {
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