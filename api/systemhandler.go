package api

import (
	"github.com/gin-gonic/gin"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/ginbase/returnfun"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/config"
	"github.com/leyle/userandrole/migrate"
	"github.com/leyle/userandrole/roleapp"
	"github.com/leyle/userandrole/userapp"
	"gopkg.in/mgo.v2/bson"
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

// 导出用户自定义 api
// 包含 item / permission / role 三部分数据
func ExportUserApiHandler(c *gin.Context, ds *dbandmq.Ds) {
	curUser, _ := GetCurUserAndRole(c)
	if curUser.IdPasswd.LoginId != userapp.AdminLoginId {
		returnfun.Return401Json(c, "不允许读取配置")
		return
	}

	if curUser == nil {
		returnfun.ReturnErrJson(c, "读取用户信息失败")
		return
	}

	db := ds.CopyDs()
	defer db.Close()

	filter := &bson.M{
		"dataFrom": bson.M{
			"$ne": roleapp.DataFromSystem,
		},
		"deleted": false,
	}


	items, err := roleapp.GetFilterItems(db, filter)
	middleware.StopExec(err)

	permissions, err := roleapp.GetFilterPermissions(db, filter)
	middleware.StopExec(err)

	roles, err := roleapp.GetFilterRoles(db, filter)
	middleware.StopExec(err)

	m := &migrate.Migrate{
		Items:       items,
		Permissions: permissions,
		Roles:       roles,
	}

	// 设置为文件下载 todo

	returnfun.ReturnOKJson(c, m)
	return
}

// 导出用户自定义的 api
func ImportUserApiHandler(c *gin.Context, ds *dbandmq.Ds) {
	curUser, _ := GetCurUserAndRole(c)
	if curUser.IdPasswd.LoginId != userapp.AdminLoginId {
		returnfun.Return401Json(c, "不允许做此操作")
		return
	}

	if curUser == nil {
		returnfun.ReturnErrJson(c, "读取用户信息失败")
		return
	}

	var form migrate.Migrate
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	db := ds.CopyDs()
	defer db.Close()

	t := util.GetCurTime()

	var insertI []interface{}
	for _, item := range form.Items {
		item.DataFrom = roleapp.DataFromUser
		item.CreateT = t
		item.UpdateT = t
		item.History = nil
		insertI = append(insertI, item)
	}

	err = db.C(roleapp.CollectionNameItem).Insert(insertI...)
	middleware.StopExec(err)

	insertI = []interface{}{}
	for _, p := range form.Permissions {
		p.DataFrom = roleapp.DataFromUser
		p.CreateT = t
		p.UpdateT = t
		p.History = nil
		insertI = append(insertI, p)
	}
	err = db.C(roleapp.CollectionNamePermission).Insert(insertI...)
	middleware.StopExec(err)

	insertI = []interface{}{}
	for _, r := range form.Roles {
		r.DataFrom = roleapp.DataFromUser
		r.CreateT = t
		r.UpdateT = t
		r.History = nil
		insertI = append(insertI, r)
	}
	err = db.C(roleapp.CollectionNameRole).Insert(insertI...)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, "")
	return
}

