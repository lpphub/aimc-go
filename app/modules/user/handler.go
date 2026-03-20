// modules/user/handler.go
package user

import (
	"aimc-go/app/server/helper"
	"aimc-go/app/server/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) register(r *gin.RouterGroup) {
	group := r.Group("/user", middleware.JwtAuth())
	{
		group.GET("/me", h.Me)
	}
}

func (h *Handler) Me(c *gin.Context) {
	userID, ok := helper.MustGetUserID(c)
	if !ok {
		return
	}

	resp, err := h.svc.Get(c, userID)
	helper.Respond(c, err, resp)
}
