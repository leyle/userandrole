package api

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/ginbase/returnfun"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/ophistory"
	"github.com/leyle/userandrole/roleapp"
)

// 新建权限
type CreatePermissionForm struct {
	Name string `json:"name" binding:"required"`
	ItemIds []string `json:"itemIds"` // 不是必选的
	Menu string `json:"menu"`
	Button string `json:"button"`
}
func CreatePermissionHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form CreatePermissionForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	// 检查名字是否存在，不加锁
	db := ds.CopyDs()
	defer db.Close()

	dbp, err := roleapp.GetPermissionByName(db, form.Name, false)
	middleware.StopExec(err)

	if dbp != nil {
		returnfun.ReturnErrJson(c, "权限已存在")
		return
	}

	permission := &roleapp.Permission{
		Id:      util.GenerateDataId(),
		Name:    form.Name,
		ItemIds: form.ItemIds,
		Menu:    form.Menu,
		Button:  form.Button,
		Deleted: false,
		CreateT: util.GetCurTime(),
	}
	permission.UpdateT = permission.CreateT

	// 操作历史记录
	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		middleware.StopExec(errors.New("获取当前用户信息失败"))
	}
	opAction := fmt.Sprintf("新建permission，名字是[%s]", form.Name)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
	permission.History = append(permission.History, opHis)
	err = roleapp.SavePermission(db, permission)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, permission)
	return
}

// 给权限再增加 item 明细
func AddItemToPermissionHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 给权限移除已有的 item 明细
func RemoteItemFromPermissionHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 修改权限基础信息
func UpdatePermissionHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 删除权限
func DeletePermissionHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 读取权限明细
func GetPermissionInfoHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 读取权限列表/搜索权限，按名字，menu，button，不支持递归搜索包含的明细
func QueryPermissionHandler(c *gin.Context, ds *dbandmq.Ds) {

}