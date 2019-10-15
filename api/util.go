package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/ginbase/returnfun"
	"github.com/leyle/userandrole/auth"
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/userandrole/roleapp"
	"github.com/leyle/userandrole/userapp"
)

// 系统配置，主要是系统校验方面
// 需要初始化这个
var AuthOption = &auth.Option{}
const AuthResult = "AUTHRESULT"

type UserOption struct {
	Ds *dbandmq.Ds
	R *redis.Client
}

// 要求所有接口都登录才行？或者说，使用这个方法的接口的，默认必须要验证的
func Auth(c *gin.Context) {
	token := c.Request.Header.Get("token")
	if token == "" {
		Logger.Error(middleware.GetReqId(c), "请求接口中无token值")
		returnfun.Return401Json(c, "No token")
		c.Next()
		return
	}

	result := auth.AuthLoginAndRole(AuthOption, token, c.Request.Method, c.Request.RequestURI, "")
	if result.Result == auth.AuthResultOK {
		c.Set(AuthResult, result)
	} else if result.Result == auth.AuthResultInValidToken {
		returnfun.Return401Json(c, "Invalid token")
		c.Next()
		return
	} else if result.Result == auth.AuthResultInValidRole {
		returnfun.Return403Json(c, "No permission")
		c.Next()
		return
	}

	c.Next()
}

func GetCurUserAndRole(c *gin.Context) (*userapp.User, []*roleapp.Role) {
	ar, exist := c.Get(AuthResult)
	if !exist {
		return nil, nil
	}
	result := ar.(*auth.AuthResult)

	return result.User, result.Role
}
