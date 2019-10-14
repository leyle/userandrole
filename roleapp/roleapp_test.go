package roleapp

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"testing"
)

func dbmgo() *dbandmq.MgoOption {
	mgo := &dbandmq.MgoOption{
		Host:     "192.168.100.233",
		Port:     "27020",
		User:     "test",
		Passwd:   "test",
		Database: "testrole",
	}

	dbandmq.InitMongodbSession(mgo)

	return mgo
}

// 初始化数据生成一些数据
func InitData() {
	mgo := dbmgo()

	db := dbandmq.NewDs(mgo)
	defer db.Close()

	// 插入 item 数据
	var itemIds []string
	item := &Item{
		Id:       util.GenerateDataId(),
		Name:     "itemA",
		Method:   "GET",
		Path:     "/api/item/a",
		Deleted:  false,
	}
	itemIds = append(itemIds, item.Id)
	db.C(CollectionNameItem).Insert(item)

	item.Id = util.GenerateDataId()
	itemIds = append(itemIds, item.Id)
	item.Name = "itemB"
	item.Method = "POST"
	db.C(CollectionNameItem).Insert(item)

	item.Id = util.GenerateDataId()
	itemIds = append(itemIds, item.Id)
	item.Name = "itemC"
	item.Method = "PUT"
	db.C(CollectionNameItem).Insert(item)

	// permissions
	var pids []string
	permission := &Permission{
		Id:      util.GenerateDataId(),
		Name:    "PA",
		ItemIds: itemIds[0:1],
		Deleted: false,
	}
	pids = append(pids, permission.Id)
	db.C(CollectionPermissionName).Insert(permission)

	permission.Id = util.GenerateDataId()
	permission.Name = "PB"
	permission.ItemIds = itemIds[0:2]
	pids = append(pids, permission.Id)
	db.C(CollectionPermissionName).Insert(permission)

	permission.Id = util.GenerateDataId()
	permission.Name = "PC"
	permission.ItemIds = itemIds[1:3]
	pids = append(pids, permission.Id)
	db.C(CollectionPermissionName).Insert(permission)

	// roles
	var roleIds []string
	role := &Role{
		Id:            util.GenerateDataId(),
		Name:          "roleA",
		PermissionIds: pids[0:1],
		Deleted:       false,
	}
	roleIds = append(roleIds, role.Id)
	db.C(CollectionNameRole).Insert(role)

	role.Id = util.GenerateDataId()
	role.Name = "roleB"
	role.PermissionIds = pids[0:2]
	roleIds = append(roleIds, role.Id)
	db.C(CollectionNameRole).Insert(role)
}


func TestGetRolesByRoleIds(t *testing.T) {
	// InitData()
	roleIds := []string{
		"5da445788543e680ee7e9cdf",
		"5da445798543e680ee7e9ce0",
	}

	mgo := dbmgo()
	db := dbandmq.NewDs(mgo)
	defer db.Close()

	roles, err := GetRolesByRoleIds(db, roleIds)
	if err != nil {
		t.Error(err)
	}

	data, err := jsoniter.MarshalToString(roles)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(data)

}

