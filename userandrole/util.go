package userandrole

import (
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/roleapp"
	"github.com/leyle/userandrole/userapp"
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
			UserId:   userId,
			RoleIds:  []string{roleapp.DefaultRoleId},
		}
	} else {
		// 所有用户都添加一个默认 roleId
		uwr.RoleIds = append(uwr.RoleIds, roleapp.DefaultRoleId)
	}

	roles, err := roleapp.GetRolesByRoleIds(db, uwr.RoleIds)
	if err != nil {
		return nil, err
	}
	uwr.Roles = roles

	return uwr, nil
}

// 初始化 admin role 和 admin account 的关系
func InitAdminWithRole(db *dbandmq.Ds) error {
	// 初始化 admin 账户
	user, err := userapp.InsureAdminAccount(db)
	if err != nil {
		return err
	}

	role, err := roleapp.InsuranceAdminRole(db)
	if err != nil {
		return err
	}

	err = initAdminWithRole(db, user.Id, user.Name, role.Id)
	if err != nil {
		Logger.Errorf("", "系统初始化admin账户，给其赋予admin role 权限失败, %s", err.Error())
		return err
	}

	Logger.Info("", "系统初始化admin，赋予adminrole权限成功")

	return nil
}

func initAdminWithRole(db *dbandmq.Ds, userId, userName, roleId string) error {
	uwr, err := GetUserWithRoleByUserId(db, userId)
	if err != nil {
		return err
	}

	if uwr == nil {
		uwr = &UserWithRole{
			Id:       util.GenerateDataId(),
			UserId:   userId,
			UserName: userName,
			Avatar:   "",
			RoleIds:  []string{roleId},
			CreateT:  util.GetCurTime(),
		}
		uwr.UpdateT = uwr.CreateT
		err = SaveUserWithRole(db, uwr)
		if err != nil {
			return err
		}
	}

	return nil
}