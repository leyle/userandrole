package api

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/ginbase/returnfun"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/ophistory"
	"github.com/leyle/userandrole/roleapp"
	"gopkg.in/mgo.v2/bson"
	"strings"
)

// 新建 item
type CreateItemForm struct {
	Name     string `json:"name" binding:"required"`
	Method   string `json:"method" binding:"required"`
	Path     string `json:"path" binding:"required"`
	Resource string `json:"resource"` // 可为空
	Menu     string `json:"menu"`
	Button   string `json:"button"`
}

func CreateItemHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form CreateItemForm
	var err error
	err = c.BindJSON(&form)
	middleware.StopExec(err)

	db := ds.CopyDs()
	defer db.Close()

	name := form.Name
	// 检查 name 是否重复，不用加锁，假设不冲突
	dbitem, err := roleapp.GetItemByName(db, name)
	middleware.StopExec(err)
	if dbitem != nil {
		Logger.Errorf(middleware.GetReqId(c), "新建role item时，已存在同名[%s]数据", name)
		returnfun.ReturnJson(c, 400, ErrCodeNameExist, "name 重复", gin.H{"id": dbitem.Id})
		return
	}

	if strings.Contains(form.Path, ":id") {
		form.Path = strings.ReplaceAll(form.Path, ":id", "*")
	}

	item := &roleapp.Item{
		Id:       util.GenerateDataId(),
		Name:     form.Name,
		Method:   strings.ToUpper(form.Method),
		Path:     form.Path,
		Resource: form.Resource,
		Menu:     form.Menu,
		Button:   form.Button,
		DataFrom: roleapp.DataFromUser,
		Deleted:  false,
		CreateT:  util.GetCurTime(),
	}
	item.UpdateT = item.CreateT

	// 记录 history 操作
	hisAction := fmt.Sprintf("新建 role item, item id[%s], item name[%s]", item.Id, item.Name)
	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		middleware.StopExec(errors.New("获取当前用户信息失败"))
	}
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, hisAction)
	if opHis != nil {
		item.History = append(item.History, opHis)
	}

	err = roleapp.SaveItem(db, item)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, item)
	return
}

// 修改item
type UpdateItemForm struct {
	Name     string `json:"name" binding:"required"`
	Method   string `json:"method" binding:"required"`
	Path     string `json:"path" binding:"required"`
	Resource string `json:"resource"` // 可为空
	Menu     string `json:"menu"`
	Button   string `json:"button"`
}

func UpdateItemHandler(c *gin.Context, ds *dbandmq.Ds) {
	var form UpdateItemForm
	var err error
	err = c.BindJSON(&form)
	middleware.StopExec(err)

	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypeItem, id) {
		returnfun.Return403Json(c, "无权修改此数据 ")
		return
	}

	db := ds.CopyDs()
	defer db.Close()

	dbitem, err := roleapp.GetItemById(db, id)
	middleware.StopExec(err)

	if dbitem == nil {
		returnfun.ReturnErrJson(c, "无指定id的数据")
		return
	}

	if strings.Contains(form.Path, ":id") {
		form.Path = strings.ReplaceAll(form.Path, ":id", "*")
	}

	dbitem.Name = form.Name
	dbitem.Method = form.Method
	dbitem.Path = form.Path
	dbitem.Resource = form.Resource
	dbitem.Deleted = false // 如果被删除过，这里就相当于重新上线
	dbitem.Menu = form.Menu
	dbitem.Button = form.Button
	dbitem.UpdateT = util.GetCurTime()

	// 记录 history 操作
	hdata, _ := jsoniter.MarshalToString(form)
	hisAction := fmt.Sprintf("修改 item，传递的数据是, %s", hdata)
	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		middleware.StopExec(errors.New("获取当前用户信息失败"))
	}
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, hisAction)
	if opHis != nil {
		dbitem.History = append(dbitem.History, opHis)
	}

	err = roleapp.UpdateItem(db, dbitem)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, dbitem)
	return
}

// 删除 item
func DeleteItemHandler(c *gin.Context, ds *dbandmq.Ds) {
	id := c.Param("id")
	if roleapp.CanNotModifyThis(roleapp.IdTypeItem, id) {
		returnfun.Return403Json(c, "无权修改信息")
		return
	}

	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		middleware.StopExec(errors.New("系统内部错误，无法识别到当前用户"))
	}

	db := ds.CopyDs()
	defer db.Close()

	err := roleapp.DeleteItemById(db, curUser.Id, curUser.Name, id)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, "")
	return
}

// 读取 item 明细
func GetItemInfoHandler(c *gin.Context, ds *dbandmq.Ds) {
	id := c.Param("id")
	db := ds.CopyDs()
	defer db.Close()

	item, err := roleapp.GetItemById(db, id)
	middleware.StopExec(err)
	returnfun.ReturnOKJson(c, item)
	return
}

// 读取 item 列表，支持分页，支持按名字/path/method搜索
func QueryItemHandler(c *gin.Context, ds *dbandmq.Ds) {
	var andCondition []bson.M

	// 过滤掉 admin
	andCondition = append(andCondition, bson.M{"name": bson.M{"$not": bson.M{"$in": roleapp.AdminItemNames}}})

	name := c.Query("name")
	if name != "" {
		andCondition = append(andCondition, bson.M{"name": bson.M{"$regex": name}})
	}

	path := c.Query("path")
	if path != "" {
		andCondition = append(andCondition, bson.M{"path": bson.M{"$regex": path}})
	}

	method := c.Query("method")
	if method != "" {
		method = strings.ToUpper(method)
		andCondition = append(andCondition, bson.M{"method": method})
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

	Q := db.C(roleapp.CollectionNameItem).Find(query)
	total, err := Q.Count()
	middleware.StopExec(err)

	var items []*roleapp.Item
	page, size, skip := util.GetPageAndSize(c)

	err = Q.Sort("-_id").Skip(skip).Limit(size).All(&items)
	middleware.StopExec(err)

	retData := gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"data":  items,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}
