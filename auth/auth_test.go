package auth

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/leyle/ginbase/dbandmq"
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

func TestAuthLoginAndRole(t *testing.T) {
	mgo := dbmgo()
	db := dbandmq.NewDs(mgo)
	defer db.Close()

	ro := &dbandmq.RedisOption{
		Host:   "192.168.100.233",
		Port:   "6380",
		Passwd: "56grTbvMYaOQ",
		DbNum:  14,
	}
	r, err := dbandmq.NewRedisClient(ro)
	if err != nil {
		t.Error(err)
	}

	opt := &Option{
		R:   r,
		Mgo: mgo,
	}

	token := "QVJNRERxQ3ZQT1pGeXlYU3YtMndMTlJfQ0lkaUZkTzZzUURPbTZTaFNGYlpTN3RzLXdyUXRwV0lzb211ZU1ZbWFDbmhDckVqSGJJNHd3eldGZG82c3c="
	method := "GET"
	uri := "/api/item/b"

	ar := AuthLoginAndRole(opt, token, method, uri, "")

	data, _ := jsoniter.MarshalToString(ar)
	fmt.Println(data)
}
