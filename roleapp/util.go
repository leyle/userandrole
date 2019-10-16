package roleapp

import (
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"sync"
)

// 根据 roleId 列表读取完整的 roles 信息
func GetRolesByRoleIds(db *dbandmq.Ds, roleIds []string) ([]*Role, error) {
	f := bson.M{
		"deleted": false,
		"_id": bson.M{
			"$in": roleIds,
		},
	}

	var roles []*Role
	err := db.C(CollectionNameRole).Find(f).All(&roles)
	if err != nil {
		Logger.Errorf("", "根据roleIds读取role信息失败, %s", err.Error())
		return nil, err
	}

	// 并行的去完善 roles 信息
	wg := sync.WaitGroup{}
	finished := make(chan bool, 1)
	errChan := make(chan error, 1)

	for _, role := range roles {
		wg.Add(1)
		go fullRole(&wg, db, role, errChan)
	}

	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case err = <-errChan:
		return nil, err
	}

	return roles, nil
}

func fullRole(wg *sync.WaitGroup, db *dbandmq.Ds, role *Role, errChan chan<- error) {
	defer wg.Done()
	ndb := db.CopyDs()
	defer ndb.Close()

	ps, err := GetPermissionsByPermissionIds(ndb, role.PermissionIds)
	if err != nil {
		errChan <- err
		return
	}
	role.Permissions = ps
}

// 根据 permissionIds 读取 permission 信息
func GetPermissionsByPermissionIds(db *dbandmq.Ds, pids []string) ([]*Permission, error) {
	f := bson.M{
		"deleted": false,
		"_id": bson.M{
			"$in": pids,
		},
	}

	var ps []*Permission
	err := db.C(CollectionPermissionName).Find(f).All(&ps)
	if err != nil {
		Logger.Errorf("", "根据permissionIds读取permission信息失败, %s", err.Error())
		return nil, err
	}

	wg := sync.WaitGroup{}
	finished := make(chan bool, 1)
	errChan := make(chan error, 1)
	for _, p := range ps {
		wg.Add(1)
		go fullPermission(&wg, db, p, errChan)
	}

	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case err = <-errChan:
		return nil, err
	}

	return ps, nil
}

func fullPermission(wg *sync.WaitGroup, db *dbandmq.Ds, permission *Permission, errChan chan<- error) {
	defer wg.Done()
	ndb := db.CopyDs()
	defer ndb.Close()

	items, err := GetItemsByItemIds(ndb, permission.ItemIds)
	if err != nil {
		errChan <- err
		return
	}
	permission.Items = items
}

// 根据 itemIds 读取 items 信息
func GetItemsByItemIds(db *dbandmq.Ds, itemIds []string) ([]*Item, error) {
	if len(itemIds) == 0 {
		return nil, nil
	}

	f := bson.M{
		"deleted": false,
		"_id": bson.M{
			"$in": itemIds,
		},
	}

	var items []*Item
	err := db.C(CollectionNameItem).Find(f).All(&items)
	if err != nil {
		Logger.Errorf("", "根据itemIds读取item信息失败, %s", err.Error())
		return nil, err
	}

	return items, nil
}

// 把 roles 的所有 item 全部抽取出来
func UnWrapRoles(roles []*Role) []*Item {
	itemMap := make(map[string]*Item)
	for _, role := range roles {
		for _, p := range role.Permissions {
			for _, item := range p.Items {
				itemMap[item.Id] = item
			}
		}
	}

	var items []*Item
	for _, item := range itemMap {
		items = append(items, item)
	}

	return items
}

// 生成一个 admin 账户 role
func InsuranceAdminRole(db *dbandmq.Ds) (*Role, error) {
	// items
	itemNames := []string{
		AdminItemName + "GET",
		AdminItemName + "POST",
		AdminItemName + "PUT",
		AdminItemName + "DELETE",
		AdminItemName + "PATCH",
		AdminItemName + "OPTION",
		AdminItemName + "HEAD",
	}

	var itemIds []string
	for _, itemName := range itemNames {
		item, err := addAdminItem(db, itemName)
		if err != nil {
			return nil, err
		}
		itemIds = append(itemIds, item.Id)
	}

	// permission
	permission, err := addAdminPermission(db, itemIds)
	if err != nil {
		return nil, err
	}

	// role
	role, err := addAdminRole(db, []string{permission.Id})
	if err != nil {
		return nil, err
	}

	Logger.Infof("", "启动roleapp，初始化adminrole成功，roleId[%s]", role.Id)
	return role, nil
}

func addAdminItem(db *dbandmq.Ds, itemName string) (*Item, error) {
	item, err := GetItemByName(db, itemName)
	if err != nil {
		return nil, err
	}

	if item == nil {
		tmp := strings.Split(itemName, ":")
		method := tmp[1]
		item = &Item{
			Id:       util.GenerateDataId(),
			Name:     itemName,
			Method:   method,
			Path:     "*",
			Resource: "*",
			Menu:     "*",
			Button:   "*",
			Deleted:  false,
			History:  nil,
			CreateT:  util.GetCurTime(),
		}
		item.UpdateT = item.CreateT

		err = SaveItem(db, item)
		if err != nil {
			return nil, err
		}
	}

	return item, nil
}

func addAdminPermission(db *dbandmq.Ds, itemIds []string) (*Permission, error) {
	permission, err := GetPermissionByName(db, AdminPermissionName, false)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		permission = &Permission{
			Id:      util.GenerateDataId(),
			Name: AdminPermissionName,
			ItemIds: itemIds,
			Menu:    "*",
			Button:  "*",
			Deleted: false,
			CreateT: util.GetCurTime(),
		}
		permission.UpdateT = permission.CreateT
		err = SavePermission(db, permission)
		if err != nil {
			return nil, err
		}
	}

	return permission, nil
}

func addAdminRole(db *dbandmq.Ds, pids []string) (*Role, error) {
	role, err := GetRoleByName(db, AdminRoleName, false)
	if err != nil {
		return nil, err
	}

	if role == nil {
		role = &Role{
			Id:            util.GenerateDataId(),
			Name:          AdminRoleName,
			PermissionIds: pids,
			Menu:          "*",
			Button:        "*",
			Deleted:       false,
			CreateT:       util.GetCurTime(),
		}
		role.UpdateT = role.CreateT
		err = SaveRole(db, role)
		if err != nil {
			return nil, err
		}
	}

	return role, nil
}

// 生成一个默认的普通role
// 这里只需要占位，后面通过接口和页面去配置注册用户的相关权限
func InsuranceDefaultRole(db *dbandmq.Ds) (*Role, error) {
	role, err := GetRoleByName(db, DefaultRoleName, false)
	if err != nil {
		return nil, err
	}

	if role == nil {
		role = &Role{
			Id:            util.GenerateDataId(),
			Name:          DefaultRoleName,
			Deleted:       false,
			CreateT:       util.GetCurTime(),
		}
		role.UpdateT = role.CreateT
		err = SaveRole(db, role)
		if err != nil {
			return nil, err
		}
	}

	// 把 role.id 赋值给默认值
	DefaultRoleId = role.Id

	Logger.Infof("", "启动roleapp，初始化普通用户role成功，roleId[%s]", role.Id)
	return role, nil
}