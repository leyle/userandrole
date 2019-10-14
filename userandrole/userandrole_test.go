package userandrole

import (
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

func TestCreateUserRole(t *testing.T) {
	uwr := &UserWithRole{
		Id:       util.GenerateDataId(),
		UserId:   "5da41d400ce239748629d9d3",
		UserName: "testUser",
		Avatar:   "",
		RoleIds: []string{"5da445788543e680ee7e9cdf"},
	}

	mgo := dbmgo()

	db := dbandmq.NewDs(mgo)
	defer db.Close()

	db.C(CollectionNameUserWithRole).Insert(uwr)
}
