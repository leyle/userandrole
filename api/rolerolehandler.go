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
	"gopkg.in/mgo.v2/bson"
	"strings"
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
		returnfun.ReturnJson(c, 400, ErrCodeNameExist, "role已存在", gin.H{"id": dbrole.Id})
		return
	}

	// 检查 pids 的有效性 todo

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
type AddPToRoleForm struct {
	Pids []string `json:"pids" binding:"required"`
}
func AddPermissionsToRoleHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form AddPToRoleForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypeRole, id) {
		returnfun.Return403Json(c, "无权做此修改")
		return
	}

	db := ds.CopyDs()
	defer db.Close()

	dbrole, err := roleapp.GetRoleById(db, id, false)
	middleware.StopExec(err)
	if dbrole == nil || dbrole.Deleted {
		returnfun.ReturnErrJson(c, "无指定id的role或role被删除")
		return
	}

	// 检查 pids 的合法性 todo

	dbrole.PermissionIds = append(dbrole.PermissionIds, form.Pids...)
	dbrole.PermissionIds = util.UniqueStringArray(dbrole.PermissionIds)
	dbrole.UpdateT = util.GetCurTime()

	// op history
	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("添加 permissiondIds %s", form.Pids)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
	dbrole.History = append(dbrole.History, opHis)

	err = roleapp.UpdateRole(db, dbrole)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, dbrole)
	return
}

// 从 role 中移除 permissions
type RemovePFromRoleForm struct {
	Pids []string `json:"pids" binding:"required"`
}
func RemovePermissionsFromRoleHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form RemovePFromRoleForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypeRole, id) {
		returnfun.Return403Json(c, "无权做此修改")
		return
	}

	db := ds.CopyDs()
	defer db.Close()

	dbrole, err := roleapp.GetRoleById(db, id, false)
	middleware.StopExec(err)

	if dbrole == nil || dbrole.Deleted {
		returnfun.ReturnErrJson(c, "无指定id的role或role被删除")
		return
	}

	// 检查 pids 的合法性
	var remainPids []string
	for _, dbpId := range dbrole.PermissionIds {
		remain := true
		for _, pid := range form.Pids {
			if dbpId == pid {
				remain = false
				break
			}
		}

		if remain {
			remainPids = append(remainPids, dbpId)
		}
	}

	dbrole.PermissionIds = remainPids
	dbrole.UpdateT = util.GetCurTime()

	// op history
	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("移除 pids %s", form.Pids)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
	dbrole.History = append(dbrole.History, opHis)

	err = roleapp.UpdateRole(db, dbrole)
	middleware.StopExec(err)
	returnfun.ReturnOKJson(c, dbrole)
	return
}

// 修改 role 信息
type UpdateRoleForm struct {
	Name string `json:"name" binding:"required"`
	Menu string `json:"menu"`
	Button string `json:"button"`
}
func UpdateRoleInfoHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form UpdateRoleForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypeRole, id) {
		returnfun.Return403Json(c, "无权做此修改")
		return
	}

	db := ds.CopyDs()
	defer db.Close()

	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("更新role信息, name[%s], menu[%s], button[%s]", form.Name, form.Menu, form.Button)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)

	update := bson.M{
		"$set": bson.M{
			"name": form.Name,
			"menu": form.Menu,
			"button": form.Button,
			"deleted": false, // 重新上线
			"updateT": util.GetCurTime(),
		},
		"$push": bson.M{
			"history": opHis,
		},
	}

	err = db.C(roleapp.CollectionNameRole).UpdateId(id, update)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, "")
	return
}

// 删除 role
// 注册用户 role 和 admin role 不能删除 todo
func DeleteRoleHandler(c *gin.Context, ds *dbandmq.Ds) {
	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypeRole, id) {
		returnfun.Return403Json(c, "无权做此修改")
		return
	}

	// op history
	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("删除role")
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)

	update := bson.M{
		"$set": bson.M{
			"deleted": true,
			"updateT": util.GetCurTime(),
		},
		"$push": bson.M{
			"history": opHis,
		},
	}

	db := ds.CopyDs()
	defer db.Close()

	err := db.C(roleapp.CollectionNameRole).UpdateId(id, update)
	middleware.StopExec(err)
	returnfun.ReturnOKJson(c, "")
	return
}

// 读取 role 明细
func GetRoleInfoHandler(c *gin.Context, ds *dbandmq.Ds) {
	id := c.Param("id")
	db := ds.CopyDs()
	defer db.Close()

	role, err := roleapp.GetRoleById(db, id, true)
	middleware.StopExec(err)
	returnfun.ReturnOKJson(c, role)
	return
}

// 搜索 role, name / menu / button
func QueryRoleHandler(c *gin.Context, ds *dbandmq.Ds) {
	var andCondition []bson.M

	// 过滤掉 admin
	andCondition = append(andCondition, bson.M{"name": bson.M{"$ne": roleapp.AdminRoleName}})

	name := c.Query("name")
	if name != "" {
		andCondition = append(andCondition, bson.M{"name": bson.M{"$regex": name}})
	}

	menu := c.Query("menu")
	if menu != "" {
		andCondition = append(andCondition, bson.M{"menu": bson.M{"$regex": menu}})
	}

	button := c.Query("button")
	if button != "" {
		andCondition = append(andCondition, bson.M{"button": bson.M{"$regex": button}})
	}

	deleted := c.Query("deleted")
	if deleted != "" {
		deleted = strings.ToUpper(deleted)
		if deleted == "TRUE" {
			andCondition = append(andCondition, bson.M{"deleted": true})
		} else {
			andCondition = append(andCondition, bson.M{"deleted": false})
		}
	}

	query := bson.M{}
	if len(andCondition) > 0 {
		query = bson.M{
			"$and": andCondition,
		}
	}

	db := ds.CopyDs()
	defer db.Close()

	Q := db.C(roleapp.CollectionNameRole).Find(query)
	total, err := Q.Count()
	middleware.StopExec(err)

	var roles []*roleapp.Role
	page, size, skip := util.GetPageAndSize(c)
	err = Q.Sort("-_id").Skip(skip).Limit(size).All(&roles)
	middleware.StopExec(err)

	retData := gin.H{
		"total": total,
		"page": page,
		"size": size,
		"data": roles,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}
