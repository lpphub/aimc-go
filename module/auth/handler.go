// module/auth/handler.go
package auth

import (
	"aimc-go/server/helper"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) register(r *gin.RouterGroup) {
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.RefreshToken)
}

func (h *Handler) Register(c *gin.Context) {
	var req AuthReq
	if !helper.MustBindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Register(c, req.Email, req.Password)
	helper.Respond(c, err, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var req AuthReq
	if !helper.MustBindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Login(c, req.Email, req.Password)
	helper.Respond(c, err, resp)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if !helper.MustBindJSON(c, &req) {
		return
	}
	resp, err := h.svc.RefreshToken(c, req.RefreshToken)
	helper.Respond(c, err, resp)
}