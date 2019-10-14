package roleapp

import (
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/ophistory"
)

// role -> permissions -> items
// 简单处理，不允许继承

// 程序启动时，初始化出来的
const DefaultRoleName = "DEFAULT:未做任何角色赋值的用户的角色"
var DefaultRoleId = ""

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

