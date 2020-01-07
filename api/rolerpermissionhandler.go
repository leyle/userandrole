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

// 新建权限
type CreatePermissionForm struct {
	Name    string   `json:"name" binding:"required"`
	ItemIds []string `json:"itemIds"` // 不是必选的
	Menu    string   `json:"menu"`
	Button  string   `json:"button"`
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
		returnfun.ReturnJson(c, 400, ErrCodeNameExist, "权限已存在", gin.H{"id": dbp.Id})
		return
	}

	permission := &roleapp.Permission{
		Id:       util.GenerateDataId(),
		Name:     form.Name,
		ItemIds:  form.ItemIds, // 不检查 itemIds 的合法性
		Menu:     form.Menu,
		Button:   form.Button,
		DataFrom: roleapp.DataFromUser,
		Deleted:  false,
		CreateT:  util.GetCurTime(),
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
type AddItemsToPermissionForm struct {
	ItemIds []string `json:"itemIds" binding:"required"`
}

func AddItemToPermissionHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form AddItemsToPermissionForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypePermission, id) {
		returnfun.Return403Json(c, "无权做此修改")
		return
	}

	db := ds.CopyDs()
	defer db.Close()

	dbp, err := roleapp.GetPermissionById(db, id, false)
	middleware.StopExec(err)
	if dbp == nil || dbp.Deleted {
		returnfun.ReturnErrJson(c, "无指定id的权限或权限已被删除")
		return
	}

	// 检查 itemIds 合法性 todo

	dbp.ItemIds = append(dbp.ItemIds, form.ItemIds...)
	dbp.ItemIds = util.UniqueStringArray(dbp.ItemIds)
	dbp.UpdateT = util.GetCurTime()

	// op history
	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("添加 itemids %s", form.ItemIds)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
	dbp.History = append(dbp.History, opHis)

	err = roleapp.UpdatePermission(db, dbp)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, dbp)
	return
}

// 给权限移除已有的 item 明细
type RemoveItemFromPermissionForm struct {
	ItemIds []string `json:"itemIds" binding:"required"`
}

func RemoveItemFromPermissionHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form RemoveItemFromPermissionForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypePermission, id) {
		returnfun.Return403Json(c, "无权做此修改")
		return
	}

	db := ds.CopyDs()
	defer db.Close()

	dbp, err := roleapp.GetPermissionById(db, id, false)
	middleware.StopExec(err)

	if dbp == nil || dbp.Deleted {
		returnfun.ReturnErrJson(c, "无指定id的权限或权限已被删除")
		return
	}

	// 检查 itemIds 合法性 todo

	var remainIds []string
	for _, dbpId := range dbp.ItemIds {
		remain := true
		for _, rid := range form.ItemIds {
			if dbpId == rid {
				remain = false
				break
			}
		}

		if remain {
			remainIds = append(remainIds, dbpId)
		}
	}

	dbp.ItemIds = remainIds
	dbp.UpdateT = util.GetCurTime()

	// op history
	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("移除 itemids %s", form.ItemIds)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
	dbp.History = append(dbp.History, opHis)

	err = roleapp.UpdatePermission(db, dbp)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, dbp)
	return

}

// 修改权限基础信息
type UpdatePermissionForm struct {
	Name   string `json:"name" binding:"required"`
	Menu   string `json:"menu"`
	Button string `json:"button"`
}

func UpdatePermissionHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form UpdatePermissionForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypePermission, id) {
		returnfun.Return403Json(c, "无权做此修改")
		return
	}

	db := ds.CopyDs()
	defer db.Close()

	// op history
	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("更新permission信息, name[%s], menu[%s], button[%s]", form.Name, form.Menu, form.Button)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)

	update := bson.M{
		"$set": bson.M{
			"name":    form.Name,
			"menu":    form.Menu,
			"button":  form.Button,
			"deleted": false, // 如果被删除过，这里相当于重新上线
			"updateT": util.GetCurTime(),
		},
		"$push": bson.M{
			"history": opHis,
		},
	}

	err = db.C(roleapp.CollectionNamePermission).UpdateId(id, update)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, "")
	return
}

// 删除权限
func DeletePermissionHandler(c *gin.Context, ds *dbandmq.Ds) {
	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypePermission, id) {
		returnfun.Return403Json(c, "无权做此修改")
		return
	}

	// op history
	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("删除权限")
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

	err := db.C(roleapp.CollectionNamePermission).UpdateId(id, update)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, "")
	return
}

// 读取权限明细
func GetPermissionInfoHandler(c *gin.Context, ds *dbandmq.Ds) {
	id := c.Param("id")
	db := ds.CopyDs()
	defer db.Close()

	p, err := roleapp.GetPermissionById(db, id, true)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, p)
	return
}

// 读取权限列表/搜索权限，按名字，menu，button，不支持递归搜索包含的明细
func QueryPermissionHandler(c *gin.Context, ds *dbandmq.Ds) {
	var andCondition []bson.M

	// 过滤掉 admin
	andCondition = append(andCondition, bson.M{"name": bson.M{"$ne": roleapp.AdminPermissionName}})

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

	Q := db.C(roleapp.CollectionNamePermission).Find(query)
	total, err := Q.Count()
	middleware.StopExec(err)

	var ps []*roleapp.Permission
	page, size, skip := util.GetPageAndSize(c)
	err = Q.Sort("-_id").Skip(skip).Limit(size).All(&ps)
	middleware.StopExec(err)

	retData := gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"data":  ps,
	}
	returnfun.ReturnOKJson(c, retData)
	return
}
