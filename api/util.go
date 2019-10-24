package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/ginbase/returnfun"
	"github.com/leyle/smsapp"
	"github.com/leyle/userandrole/auth"
	"github.com/leyle/userandrole/roleapp"
	"github.com/leyle/userandrole/userapp"
)

// 系统配置，主要是系统校验方面
// 需要初始化这个
var AuthOption = &auth.Option{} // 调用本包，需要给这个变量赋值
const AuthResultCtxKey = "AUTHRESULT"

type UserOption struct {
	Ds *dbandmq.Ds
	R *redis.Client
	WeChatOpt map[string]*userapp.WeChatOption // 微信配置， key 是平台
	PhoneOpt *smsapp.SmsOption // phone 发送配置
}

// 要求所有接口都登录才行？或者说，使用这个方法的接口的，默认必须要验证的
func Auth(c *gin.Context) {
	token := c.Request.Header.Get("token")
	if token == "" {
		Logger.Error(middleware.GetReqId(c), "请求接口中无token值")
		returnfun.Return401Json(c, "No token")
		return
	}

	result := auth.AuthLoginAndRole(AuthOption, token, c.Request.Method, c.Request.URL.Path, "")
	debugPrintUserRoleInfo(c, result)

	if result.Result == auth.AuthResultOK {
		c.Set(AuthResultCtxKey, result)
	} else if result.Result == auth.AuthResultInValidToken {
		returnfun.Return401Json(c, "Invalid token")
		return
	} else if result.Result == auth.AuthResultInValidRole {
		returnfun.Return403Json(c, "No permission")
		return
	}

	c.Next()
}

func GetCurUserAndRole(c *gin.Context) (*userapp.User, []*roleapp.Role) {
	ar, exist := c.Get(AuthResultCtxKey)
	if !exist {
		return nil, nil
	}
	result := ar.(*auth.AuthResult)

	return result.User, result.Roles
}

func debugPrintUserRoleInfo(c *gin.Context, result *auth.AuthResult) {
	if result.User == nil {
		return
	}

	if result.Roles == nil {
		Logger.Debugf(middleware.GetReqId(c), "用户[%s][%s]无任何权限", result.User.Id, result.User.Name)
		return
	}

	var names []string
	for _, role := range result.Roles {
		names = append(names, role.Name)
	}

	Logger.Debugf(middleware.GetReqId(c), "用户[%s][%s]包含的角色为 %s",result.User.Id, result.User.Name, names)
}