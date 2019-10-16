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
	"github.com/leyle/userandrole/userandrole"
)

// uwr means user with role

// 给用户添加 roles
type AddRolesToUserForm struct {
	UserId string `json:"userId" binding:"required"`
	UserName string `json:"userName"`
	Avatar string `json:"avatar"`
	RoleIds []string `json:"roleIds" binding:"required"`
}
func AddRolesToUserHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form AddRolesToUserForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	db := ds.CopyDs()
	defer db.Close()

	// 检查所有的 roleids 的有效性 todo

	// 不用锁定数据，低频操作

	// 检查 uwr 是否存在，不存在新建，存在就是更新
	uwr, err := userandrole.GetUserWithRoleByUserId(db, form.UserId)
	middleware.StopExec(err)
	if uwr == nil {
		uwr = &userandrole.UserWithRole{
			Id:       util.GenerateDataId(),
			UserId:   form.UserId,
			UserName: form.UserName,
			Avatar:   form.Avatar,
			RoleIds:  form.RoleIds,
			CreateT:  util.GetCurTime(),
		}
		uwr.UpdateT = uwr.CreateT
	} else {
		uwr.RoleIds = append(uwr.RoleIds, form.RoleIds...)
	}
	uwr.RoleIds = util.UniqueStringArray(uwr.RoleIds)

	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		middleware.StopExec(errors.New("获取当前用户信息失败"))
		return
	}
	opAction := fmt.Sprintf("给用户[%s][%s]添加roleIds[%s]", form.UserId, form.UserName, form.RoleIds)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
	uwr.History = append(uwr.History, opHis)

	err = userandrole.SaveUserWithRole(db, uwr)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, uwr)
	return
}

// 取消 用户的某些 roles
func RemoveRolesFromUserHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 读取指定 userid 的 roles 信息
func GetUserRolesHandler(c *gin.Context, ds *dbandmq.Ds) {

}

// 读取授权了的用户列表
func QueryUWRHandler(c *gin.Context, ds *dbandmq.Ds) {

}
