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

// 新建 role
type CreateRoleForm struct {
	Name string `json:"name" binding:"required"`
	Pids []string `json:"pids"` // 可以没有值
	Menu string `json:"menu"`
	Button string `json:"button"`
}
func CreateRoleHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form CreateRoleForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	db := ds.CopyDs()
	defer db.Close()

	// 检查 name 是否存在
	dbrole, err := roleapp.GetRoleByName(db, form.Name, false)
	middleware.StopExec(err)
	if dbrole != nil {
		returnfun.ReturnErrJson(c, "role已存在")
		return
	}

	role := &roleapp.Role{
		Id:            util.GenerateDataId(),
		Name:          form.Name,
		PermissionIds: form.Pids,
		Menu:          form.Menu,
		Button:        form.Button,
		Deleted:       false,
		CreateT:       util.GetCurTime(),
	}
	role.UpdateT = role.CreateT

	// opHistory
	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		middleware.StopExec(errors.New("读取当前用户信息失败"))
	}
	hisAction := fmt.Sprintf("新建 role, role name[%s], pids[%s], menu[%s], button[%s]", form.Name, form.Pids, form.Menu, form.Button)

	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, hisAction)
	role.History = append(role.History, opHis)

	err = roleapp.SaveRole(db, role)
	middleware.StopExec(err)
	returnfun.ReturnOKJson(c, role)
	return
}

// 给 role 添加 permission
func AddPermissionsToRoleHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 从 role 中移除 permissions
func RemovePermissionsFromRoleHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 修改 role 信息
func UpdateRoleInfoHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 删除 role
func DeleteRoleHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 读取 role 明细
func GetRoleInfoHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 搜索 role
func QueryRoleHandler(c *gin.Context, ds *dbandmq.Ds) {

}
