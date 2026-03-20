// modules/user/init.go
package user

import (
	"aimc-go/app/modules/core"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var _ core.Module = (*Module)(nil)

type Module struct {
	Service *Service
	handler *Handler
}

func New(db *gorm.DB) *Module {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	return &Module{
		Service: svc,
		handler: h,
	}
}

func (m *Module) Routes(r *gin.RouterGroup) {
	m.handler.register(r)
}
