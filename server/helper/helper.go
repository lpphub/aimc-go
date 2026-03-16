package helper

import (
	"strconv"
	"aimc-go/server/middleware"
	"aimc-go/shared/errs"

	"github.com/gin-gonic/gin"
	"github.com/lpphub/goweb/base"
	"github.com/lpphub/goweb/pkg/logging"
)

func MustBindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		logging.Errorw(c.Request.Context(), err)
		base.Fail(c, errs.ErrInvalidParam)
		return false
	}
	return true
}

func MustBindQuery(c *gin.Context, obj any) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		logging.Errorw(c.Request.Context(), err)
		base.Fail(c, errs.ErrInvalidParam)
		return false
	}
	return true
}

func MustGetUserID(c *gin.Context) (uint, bool) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		base.Fail(c, errs.ErrNoToken)
		return 0, false
	}
	return userID, true
}

func MustParseUintParam(c *gin.Context, param string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(param), 10, 32)
	if err != nil {
		logging.Errorw(c, err)
		base.Fail(c, errs.ErrInvalidParam)
		return 0, false
	}
	return uint(id), true
}

func Respond(c *gin.Context, err error, data ...any) {
	base.Respond(c, err, data...)
}