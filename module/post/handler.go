// module/post/handler.go
package post

import (
	"strconv"

	"aimc-go/server/helper"
	"aimc-go/server/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) register(r *gin.RouterGroup) {
	group := r.Group("/posts")
	{
		group.GET("", h.List)
		group.GET("/:id", h.Get)
		group.POST("", middleware.JwtAuth(), h.Create)
		group.PUT("/:id", middleware.JwtAuth(), h.Update)
		group.DELETE("/:id", middleware.JwtAuth(), h.Delete)
	}
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := helper.MustGetUserID(c)
	if !ok {
		return
	}

	var req CreatePostReq
	if !helper.MustBindJSON(c, &req) {
		return
	}

	resp, err := h.svc.Create(c, userID, req)
	helper.Respond(c, err, resp)
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	list, total, err := h.svc.List(c, page, pageSize)
	helper.Respond(c, err, gin.H{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := helper.MustParseUintParam(c, "id")
	if !ok {
		return
	}

	resp, err := h.svc.Get(c, id)
	helper.Respond(c, err, resp)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := helper.MustParseUintParam(c, "id")
	if !ok {
		return
	}

	var req UpdatePostReq
	if !helper.MustBindJSON(c, &req) {
		return
	}

	err := h.svc.Update(c, id, req)
	helper.Respond(c, err, gin.H{"message": "updated successfully"})
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := helper.MustParseUintParam(c, "id")
	if !ok {
		return
	}

	err := h.svc.Delete(c, id)
	helper.Respond(c, err, gin.H{"message": "deleted successfully"})
}