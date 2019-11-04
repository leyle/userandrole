package util

import (
	"errors"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/roleapp"
	. "github.com/leyle/ginbase/consolelog"
)

// 初始化所有的 api item 到数据库中
func RbacHelper(db *dbandmq.Ds) error {
	var err error
	t := util.GetCurTime()

	Logger.Debug("", "开始初始化 role items...")

	// role items
	defaultRoleItems := []*roleapp.Item {
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "用户修改自己的账户密码",
			Method:   "POST",
			Path:     "/api/user/idpasswd/changepasswd",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "微信绑定手机号",
			Method:   "POST",
			Path:     "/api/user/wx/bindphone",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "用户读取自身信息",
			Method:   "GET",
			Path:     "/api/user/me",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "退出登录",
			Method:   "GET",
			Path:     "/api/user/logout",
		},
	}

	roleItems := []*roleapp.Item{
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "新建item",
			Method:   "POST",
			Path:     "/api/role/item",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "修改item",
			Method:   "PUT",
			Path:     "/api/role/item/:id",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "删除item",
			Method:   "DELETE",
			Path:     "/api/role/item/:id",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "读取item明细",
			Method:   "GET",
			Path:     "/api/role/item/:id",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "搜索item",
			Method:   "GET",
			Path:     "/api/role/items",
		},

		//////////////////////////////////// permission
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "新建permission",
			Method:   "POST",
			Path:     "/api/role/permission",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "给permission添加items",
			Method:   "POST",
			Path:     "/api/role/permission/*/additems",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "取消permission某些items",
			Method:   "POST",
			Path:     "/api/role/permission/*/delitems",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "修改permission基本信息",
			Method:   "PUT",
			Path:     "/api/role/permission/*",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "删除permission",
			Method:   "DELETE",
			Path:     "/api/role/permission/*",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "读取permission信息",
			Method:   "GET",
			Path:     "/api/role/permission/*",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "搜索permission",
			Method:   "GET",
			Path:     "/api/role/permissions",
		},

		///////////////////////////////
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "新建role",
			Method:   "POST",
			Path:     "/api/role/role",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "给role添加权限",
			Method:   "POST",
			Path:     "/api/role/role/*/addps",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "取消role某些权限",
			Method:   "POST",
			Path:     "/api/role/role/*/delps",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "修改role基本信息",
			Method:   "PUT",
			Path:     "/api/role/role/*",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "删除role",
			Method:   "DELETE",
			Path:     "/api/role/role/*",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "给role添加childrole",
			Method:   "POST",
			Path:     "/api/role/role/*/addchildrole",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "删除role某些childrole",
			Method:   "POST",
			Path:     "/api/role/role/*/delchildrole",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "查看role信息",
			Method:   "GET",
			Path:     "/api/role/role/*",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "搜索role",
			Method:   "GET",
			Path:     "/api/role/roles",
		},

		/////////////////////////////////////
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "替用户创建账户密码",
			Method:   "POST",
			Path:     "/api/user/idpasswd",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "替用户创建手机号登录账户",
			Method:   "POST",
			Path:     "/api/user/phone",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "封禁用户",
			Method:   "POST",
			Path:     "/api/user/ban",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "解禁用户",
			Method:   "POST",
			Path:     "/api/user/unban",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "替用户重置密码",
			Method:   "POST",
			Path:     "/api/user/idpasswd/resetpasswd",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "读取指定id的用户信息",
			Method:   "GET",
			Path:     "/api/user/user/*",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "查看指定id用户登录历史",
			Method:   "GET",
			Path:     "/api/user/loginhistory/*",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "搜索用户列表",
			Method:   "GET",
			Path:     "/api/user/users",
		},

		///////////////////////////////////////////
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "给用户添加角色",
			Method:   "POST",
			Path:     "/api/uwr/addroles",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "取消用户角色",
			Method:   "POST",
			Path:     "/api/uwr/delroles",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "读取指定id用户的roles信息",
			Method:   "GET",
			Path:     "/api/uwr/user/*",
		},
		&roleapp.Item{
			Id:       util.GenerateDataId(),
			Name:     "搜索已授权用户列表",
			Method:   "GET",
			Path:     "/api/uwr/users",
		},
	}

	// 1. 检查上面所有的 items 是否存在，不存在的就创建
	var dris []*roleapp.Item
	for _, item := range defaultRoleItems {
		item.CreateT = t
		item.UpdateT = t
		item, err = insurenItem(db, item)
		dris = append(dris, item)
		if err != nil {
			return err
		}
	}

	var ris []*roleapp.Item
	for _, item := range roleItems {
		item.CreateT = t
		item.UpdateT = t
		item, err = insurenItem(db, item)
		if err != nil {
			return err
		}
		ris = append(ris, item)
	}

	// 2. 将 default item 附给 default role id
	var ditemIds []string
	for _, di := range dris {
		ditemIds = append(ditemIds, di.Id)
	}
	Logger.Debugf("", "默认 role item ids, %s", ditemIds)

	defaultP := &roleapp.Permission{
		Id:      util.GenerateDataId(),
		Name:    "注册用户默认权限",
		ItemIds: ditemIds,
		Deleted: false,
		CreateT: t,
		UpdateT: t,
	}
	defaultP, err = insurePermission(db, defaultP)
	if err != nil {
		return err
	}

	// 3. 将所有的 item 给 api 管理权限
	var allItemIds []string
	for _, item := range ris {
		allItemIds = append(allItemIds, item.Id)
	}
	allItemIds = append(allItemIds, ditemIds...)
	apiP := &roleapp.Permission{
		Id:      util.GenerateDataId(),
		Name:    "roleApi 管理权限",
		ItemIds: allItemIds,
		Deleted: false,
		CreateT: t,
		UpdateT: t,
	}

	apiP, err = insurePermission(db, apiP)
	if err != nil {
		return err
	}

	// 4. 将 defautp 给 default role
	defaultRole, err := roleapp.GetRoleById(db, roleapp.DefaultRoleId, false)
	if err != nil {
		return err
	}
	if defaultRole == nil {
		Logger.Error("", "发生错误，不存在默认用户role")
		return errors.New("未初始化默认注册用户角色")
	}
	defaultRole.PermissionIds = append(defaultRole.PermissionIds, defaultP.Id)
	defaultRole.PermissionIds = util.UniqueStringArray(defaultRole.PermissionIds)
	err = roleapp.UpdateRole(db, defaultRole)
	if err != nil {
		return err
	}

	// 5. 将所有 p 给 api role
	apiR := &roleapp.Role{
		Id:            util.GenerateDataId(),
		Name:          "api管理员",
		PermissionIds: allItemIds,
		Deleted:       false,
		CreateT:       t,
		UpdateT:       t,
	}
	apiRole, err := roleapp.GetRoleByName(db, apiR.Name, false)
	if err != nil {
		return err
	}
	if apiRole == nil {
		err = roleapp.SaveRole(db, apiR)
	} else {
		apiRole.PermissionIds = append(apiRole.PermissionIds, apiP.Id)
		apiRole.PermissionIds = util.UniqueStringArray(apiRole.PermissionIds)
		err = roleapp.UpdateRole(db, apiRole)
		if err != nil {
			return err
		}
	}

	Logger.Debug("", "初始化 role items 结束")
	return nil
}

func insurenItem(db *dbandmq.Ds, item *roleapp.Item) (*roleapp.Item, error) {
	// 按 name 查询，不存在就创建
	dbitem, err := roleapp.GetItemByName(db, item.Name)
	if err != nil {
		return nil, err
	}
	if dbitem != nil {
		return dbitem, nil
	}

	Logger.Debugf("", "item[%s][%s][%s]不存在， 准备创建", item.Name, item.Method, item.Path)
	err = roleapp.SaveItem(db, item)
	return item, err
}

func insurePermission(db *dbandmq.Ds, p *roleapp.Permission) (*roleapp.Permission, error) {
	Logger.Debugf("", "当前处理权限[%s]", p.Name)
	dbp, err := roleapp.GetPermissionByName(db, p.Name, false)
	if err != nil {
		return nil, err
	}
	if dbp == nil {
		Logger.Debugf("", "permission [%s]不存在，准备创建", p.Name)
		err = roleapp.SavePermission(db, p)
		if err != nil {
			return nil, err
		}
		return p, nil
	}

	dbp.ItemIds = append(dbp.ItemIds, p.ItemIds...)
	dbp.ItemIds = util.UniqueStringArray(dbp.ItemIds)
	dbp.UpdateT = p.UpdateT
	err = roleapp.UpdatePermission(db, dbp)
	if err != nil {
		return nil, err
	}
	return dbp, nil
}

