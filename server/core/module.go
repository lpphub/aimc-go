package core

import "github.com/gin-gonic/gin"

type Module interface {
	Routes(r *gin.RouterGroup)
}