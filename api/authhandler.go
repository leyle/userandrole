package api

import (
	"github.com/gin-gonic/gin"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/ginbase/returnfun"
	"github.com/leyle/userandrole/auth"
)

// 提供 auth 接口给调用者进行认证
type AuthForm struct {
	Token string `json:"token" binding:"required"`
	Method string `json:"method" binding:"required"`
	Path string `json:"path" binding:"required"`
}

func AuthHandler(c *gin.Context, uo *UserOption) {
	var form AuthForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	option := &auth.Option{
		R:  uo.R,
		Ds: uo.Ds,
	}

	result := auth.AuthLoginAndRole(option, form.Token, form.Method, form.Path, "")

	returnfun.ReturnOKJson(c, result)
	return
}
