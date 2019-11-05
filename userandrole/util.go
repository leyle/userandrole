package userandrole

import (
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/roleapp"
	"github.com/leyle/userandrole/userapp"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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
		Logger.Debugf("", "用户[%s]无任何角色，准备分配默认角色", userId)
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

	roles, err := roleapp.GetRolesByRoleIds(db, uwr.RoleIds, true)
	if err != nil {
		return nil, err
	}
	uwr.Roles = roles

	// 扩展展开 menus 和 buttons
	menus, buttons := UnwrapRoleMenuAndButton(roles)
	uwr.Menus = menus
	uwr.Buttons = buttons

	// 展开所有的角色
	var childrenRole []*roleapp.ChildRole
	for _, role := range roles {
		if len(role.ChildrenRoles) > 0 {
			childrenRole = append(childrenRole, role.ChildrenRoles...)
		}
	}
	if len(childrenRole) > 0 {
		childrenRole = uniqueChildrenRole(childrenRole)
		uwr.ChildrenRole = childrenRole
	}

	return uwr, nil
}

func uniqueChildrenRole(childrenRole []*roleapp.ChildRole) []*roleapp.ChildRole {
	roleMap := make(map[string]*roleapp.ChildRole)
	for _, cr := range childrenRole {
		roleMap[cr.Id] = cr
	}

	var ret []*roleapp.ChildRole
	for _, v := range roleMap {
		ret = append(ret, v)
	}
	return ret
}

// 展开所有的 menus 和 buttons，包含了 role / permission / item 上的数据，去重
func UnwrapRoleMenuAndButton(roles []*roleapp.Role) ([]string, []string) {
	var menus []string
	var buttons []string
	for _, role := range roles {
		if role.Menu != "" {
			menus = append(menus, role.Menu)
		}
		if role.Button != "" {
			buttons = append(buttons, role.Button)
		}

		for _, p := range role.Permissions {
			if p.Menu != "" {
				menus = append(menus, p.Menu)
			}
			if p.Button != "" {
				buttons = append(buttons, p.Button)
			}

			for _, item := range p.Items {
				if item.Menu != "" {
					menus = append(menus, item.Menu)
				}
				if item.Button != "" {
					buttons = append(buttons, item.Button)
				}
			}
		}
	}

	return menus, buttons
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
		err = SaveUserWithRole(db, uwr, false)
		if err != nil {
			return err
		}
	}

	return nil
}