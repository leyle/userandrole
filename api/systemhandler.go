package api

import (
	"github.com/gin-gonic/gin"
	"github.com/leyle/ginbase/returnfun"
	"github.com/leyle/userandrole/config"
	"github.com/leyle/userandrole/userapp"
)

// 读取返回 mongodb 和 redis 的配置
// 仅 admin 账户能够读取数据
func GetMongodbAndRedisConfHandler(c *gin.Context, conf *config.Config) {
	curUser, _ := GetCurUserAndRole(c)
	if curUser.IdPasswd.LoginId != userapp.AdminLoginId {
		returnfun.Return401Json(c, "不允许读取配置")
		return
	}

	if curUser == nil {
		returnfun.ReturnErrJson(c, "读取用户信息失败")
		return
	}

	retData := gin.H{
		"redis": conf.Redis,
		"mongodb": conf.Mongodb,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}
