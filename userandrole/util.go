package userandrole

import (
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/roleapp"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	. "github.com/leyle/ginbase/consolelog"
)

func GetUserRoles(db *dbandmq.Ds, userId string) (*UserWithRole, error) {
	f := bson.M{
		"userId": userId,
	}

	var uwr *UserWithRole
	err := db.C(CollectionNameUserWithRole).Find(f).One(&uwr)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据用户[%s]读取角色关联表失败, %s", userId, err.Error())
		return nil, err
	}

	if uwr == nil {
		// 用户没有任何授权，返回默认角色
		uwr = &UserWithRole{
			Id:       util.GenerateDataId(),
			UserId:   "",
			UserName: "",
			Avatar:   "",
			RoleIds:  []string{roleapp.DefaultRoleId},
		}
	}

	roles, err := roleapp.GetRolesByRoleIds(db, uwr.RoleIds)
	if err != nil {
		return nil, err
	}
	uwr.Roles = roles

	return uwr, nil
}
