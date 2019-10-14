package userandrole

import (
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/ophistory"
	"github.com/leyle/userandrole/roleapp"
)

// 用户角色关联
const CollectionNameUserWithRole = "userWithRole"
type UserWithRole struct {
	Id string `json:"id" bson:"_id"`
	UserId string `json:"userId" bson:"userId"`
	UserName string `json:"userName" bson:"userName"`
	Avatar string `json:"avatar" bson:"avatar"`
	RoleIds []string `json:"-" bson:"roleIds"`
	Roles []*roleapp.Role `json:"roles" bson:"-"`

	History []*ophistory.OperationHistory `json:"history" bson:"history"`

	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}