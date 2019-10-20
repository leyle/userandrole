package userandrole

import (
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/ophistory"
	"github.com/leyle/userandrole/roleapp"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 用户角色关联
const CollectionNameUserWithRole = "userWithRole"
var IKUserWithRole = &dbandmq.IndexKey{
	Collection:    CollectionNameUserWithRole,
	SingleKey:     []string{"userId", "roleIds"},
	UniqueKey:     []string{"userId"},
}
type UserWithRole struct {
	Id string             `json:"id" bson:"_id"`
	UserId string         `json:"userId" bson:"userId"`
	UserName string       `json:"userName" bson:"userName"`
	Avatar string         `json:"avatar" bson:"avatar"`
	RoleIds []string      `json:"-" bson:"roleIds"`
	Roles []*roleapp.Role `json:"roles" bson:"-"`

	History []*ophistory.OperationHistory `json:"history" bson:"history"`

	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// 根据 userId 查询 userwithrole
func GetUserWithRoleByUserId(db *dbandmq.Ds, userId string) (*UserWithRole, error) {
	f := bson.M{
		"userId": userId,
	}

	var uwr *UserWithRole
	err := db.C(CollectionNameUserWithRole).Find(f).One(&uwr)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据userId[%s]查询userWithRole失败, %s", userId, err.Error())
		return nil, err
	}

	return uwr, nil
}

func SaveUserWithRole(db *dbandmq.Ds, uwr *UserWithRole, update bool) error {
	if update {
		return UpdateUserWithRole(db, uwr)
	}
	return db.C(CollectionNameUserWithRole).Insert(uwr)
}

func UpdateUserWithRole(db *dbandmq.Ds, uwr *UserWithRole) error {
	return db.C(CollectionNameUserWithRole).UpdateId(uwr.Id, uwr)
}