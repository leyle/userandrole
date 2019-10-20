package roleapp

import (
	"fmt"
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/ophistory"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// role -> permissions -> items
// 简单处理，不允许继承

// 程序启动时，初始化出来的
const DefaultRoleName = "注册用户默认角色"
var DefaultRoleId = ""

const (
	AdminRoleName = "admin"
	AdminPermissionName = "admin"
	AdminItemName = "admin:"
)

// 定义不可修改的 item / permissid / role Id
const (
	IdTypeItem = "item"
	IdTypePermission = "permission"
	IdTypeRole = "role"
)
var (
	CanNotModifyItemIds []string
	CanNotModifyPermissionIds []string
	CanNotMidifyRoleIds []string
)

var AdminItemNames = []string{
	AdminItemName + "GET",
	AdminItemName + "POST",
	AdminItemName + "PUT",
	AdminItemName + "DELETE",
	AdminItemName + "PATCH",
	AdminItemName + "OPTION",
	AdminItemName + "HEAD",
}

// item
const CollectionNameItem = "item"
type Item struct {
	Id string `json:"id" bson:"_id"`
	Name string `json:"name" bson:"name"`
	// api
	Method string `json:"method" bson:"method"`
	Path string `json:"path" bson:"path"`
	Resource string `json:"resource" bson:"resource"` // 可以为空

	// html
	Menu string `json:"menu" bson:"menu"`
	Button string `json:"button" bson:"button"`

	Deleted bool `json:"deleted" bson:"deleted"`

	History []*ophistory.OperationHistory `json:"history" bson:"history"`

	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// permission
const CollectionPermissionName = "permission"
type Permission struct {
	Id string `json:"id" bson:"_id"`
	Name string `json:"name" bson:"name"`

	ItemIds []string `json:"-" bson:"itemIds"`
	Items []*Item `json:"items" bson:"-"`

	// html
	Menu string `json:"menu" bson:"menu"`
	Button string `json:"button" bson:"button"`

	Deleted bool `json:"deleted" bson:"deleted"`
	History []*ophistory.OperationHistory `json:"history" bson:"history"`

	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// role
const CollectionNameRole = "role"
type Role struct {
	Id string `json:"id" bson:"_id"`
	Name string `json:"name" bson:"name"`

	PermissionIds []string `json:"-" bson:"permissionIds"`
	Permissions []*Permission `json:"permissions" bson:"-"`

	// html
	Menu string `json:"menu" bson:"menu"`
	Button string `json:"button" bson:"button"`

	Deleted bool `json:"deleted" bson:"deleted"`
	History []*ophistory.OperationHistory `json:"history" bson:"history"`

	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// 根据 id 读取 item
func GetItemById(db *dbandmq.Ds, id string) (*Item, error) {
	var item *Item
	err := db.C(CollectionNameItem).FindId(id).One(&item)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据id[%s]读取 item 信息失败, %s", id, err.Error())
		return nil, err
	}
	return item, nil
}

// 根据 name 读取 item
func GetItemByName(db *dbandmq.Ds, name string) (*Item, error) {
	f := bson.M{
		"name": name,
	}

	var item *Item
	err := db.C(CollectionNameItem).Find(f).One(&item)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据name[%s]读取 role item 失败, %s", err.Error())
		return nil,  err
	}

	return item, nil
}

// 存储 item
func SaveItem(db *dbandmq.Ds, item *Item) error {
	return db.C(CollectionNameItem).Insert(item)
}

// 更新指定 id 的 item
func UpdateItem(db *dbandmq.Ds, item *Item) error {
	err := db.C(CollectionNameItem).UpdateId(item.Id, item)
	return err
}

// 删除指定 id 的 item
// 不需要单独的去删除包含了自己的 permission 中的数据
// permission 中会标记这个数据，并且不做显示
func DeleteItemById(db *dbandmq.Ds, userId, userName, id string) error {
	opAction := fmt.Sprintf("删除 item, itemId[%s]", id)
	opHis := ophistory.NewOpHistory(userId, userName, opAction)

	update := bson.M{
		"$set": bson.M{
			"deleted": true,
			"updateT": util.GetCurTime(),
		},
		"$push": bson.M{
			"history": opHis,
		},
	}

	err := db.C(CollectionNameItem).UpdateId(id, update)
	if err != nil {
		Logger.Errorf("", "删除item[%s]失败,%s", id, err.Error())
		return err
	}
	return nil
}

// 根据 name 读取 permission
func GetPermissionByName(db *dbandmq.Ds, name string, more bool) (*Permission, error) {
	f := bson.M{
		"name": name,
	}

	var p *Permission
	err := db.C(CollectionPermissionName).Find(f).One(&p)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据permission name[%s]读取permission信息失败, %s", name, err.Error())
		return nil, err
	}

	if p == nil {
		return nil, nil
	}

	if more {
		items, err := GetItemsByItemIds(db, p.ItemIds)
		if err == nil {
			p.Items = items
		}
	}

	return p, nil
}

func GetPermissionById(db *dbandmq.Ds, id string, more bool) (*Permission, error) {
	var p *Permission
	err := db.C(CollectionPermissionName).FindId(id).One(&p)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据 permission id[%s]读取permission信息失败, %s", id, err.Error())
		return nil, err
	}

	if p == nil {
		return nil, nil
	}

	if more {
		items, err := GetItemsByItemIds(db, p.ItemIds)
		if err == nil {
			p.Items = items
		}
	}

	return p, nil
}

// 存储 permission
func SavePermission(db *dbandmq.Ds, p *Permission) error {
	return db.C(CollectionPermissionName).Insert(p)
}

func UpdatePermission(db *dbandmq.Ds, p *Permission) error {
	return db.C(CollectionPermissionName).UpdateId(p.Id, p)
}

// 根据 name 读取 role
func GetRoleByName(db *dbandmq.Ds, name string, more bool) (*Role, error) {
	f := bson.M{
		"name": name,
	}

	var role *Role
	err := db.C(CollectionNameRole).Find(f).One(&role)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据role name[%s]读取role信息失败, %s", name, err.Error())
		return nil, err
	}

	if role == nil {
		return nil, nil
	}

	if more {
		ps, err := GetPermissionsByPermissionIds(db, role.PermissionIds)
		if err == nil {
			role.Permissions = ps
		}
	}

	return role, nil
}

func GetRoleById(db *dbandmq.Ds, id string, more bool) (*Role, error) {
	var role *Role
	err := db.C(CollectionNameRole).FindId(id).One(&role)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据role id[%s]读取role信息失败, %s", id, err.Error())
		return nil, err
	}

	if role == nil {
		return nil, nil
	}

	if more {
		ps, err := GetPermissionsByPermissionIds(db, role.PermissionIds)
		if err == nil {
			role.Permissions = ps
		}
	}

	return role, nil
}

func SaveRole(db *dbandmq.Ds, role *Role) error {
	return db.C(CollectionNameRole).Insert(role)
}

func UpdateRole(db *dbandmq.Ds, role *Role) error {
	return db.C(CollectionNameRole).UpdateId(role.Id, role)
}