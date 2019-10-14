package roleapp

import (
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"gopkg.in/mgo.v2/bson"
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