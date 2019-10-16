package main

import (
	"fmt"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/userandrole/api"
	. "github.com/leyle/userandrole/auth"
	"github.com/leyle/userandrole/userandrole"
	"os"
)

func main() {
	ro := &dbandmq.RedisOption{
		Host:   "192.168.100.233",
		Port:   "6380",
		Passwd: "56grTbvMYaOQ",
		DbNum:  14,
	}
	rClient, err := dbandmq.NewRedisClient(ro)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mgo := &dbandmq.MgoOption{
		Host:     "192.168.100.233",
		Port:     "27020",
		User:     "test",
		Passwd:   "test",
		Database: "testrole",
	}

	db := dbandmq.NewDs(mgo)
	defer db.Close()

	err = userandrole.InitAdminWithRole(db)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	authOption := &Option{
		R:   rClient,
		Mgo: mgo,
	}

	userOption := &api.UserOption{
		Ds: db,
		R:  rClient,
	}

	api.AuthOption = authOption

	r := middleware.SetupGin()
	apiRouter := r.Group("/api")

	api.RoleRouter(db, apiRouter.Group(""))

	api.UserRouter(userOption, apiRouter.Group(""))

	api.UserWithRoleRouter(db, apiRouter.Group(""))

	addr := "0.0.0.0:9300"
	err = r.Run(addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
