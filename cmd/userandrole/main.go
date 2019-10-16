package main

import (
	"flag"
	"fmt"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/userandrole/api"
	. "github.com/leyle/userandrole/auth"
	"github.com/leyle/userandrole/config"
	"github.com/leyle/userandrole/roleapp"
	"github.com/leyle/userandrole/userandrole"
	"os"
)

func main() {
	var err error
	var port string
	var cfile string

	flag.StringVar(&port, "p", "", "-p 9300")
	flag.StringVar(&cfile, "c", "", "-c /path/to/config/file")
	flag.Parse()
	if cfile == "" {
		fmt.Println("缺少运行的配置文件")
		os.Exit(1)
	}

	conf, err := config.LoadConf(cfile)
	if err != nil {
		os.Exit(1)
	}
	conf.Server.Port = port

	ro := &dbandmq.RedisOption{
		Host:   conf.Redis.Host,
		Port:   conf.Redis.Port,
		Passwd: conf.Redis.Passwd,
		DbNum:  conf.Redis.DbNum,
	}
	rClient, err := dbandmq.NewRedisClient(ro)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mgo := &dbandmq.MgoOption{
		Host:     conf.Mongodb.Host,
		Port:    conf.Mongodb.Port,
		User:     conf.Mongodb.User,
		Passwd:   conf.Mongodb.Passwd,
		Database: conf.Mongodb.Database,
	}

	db := dbandmq.NewDs(mgo)
	defer db.Close()

	// 初始化 admin 和相关权限
	err = userandrole.InitAdminWithRole(db)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 初始化普通用户角色
	_, err = roleapp.InsuranceDefaultRole(db)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 初始化验证相关需要的配置
	authOption := &Option{
		R:   rClient,
		Mgo: mgo,
	}
	api.AuthOption = authOption

	r := middleware.SetupGin()
	apiRouter := r.Group("/api")

	// 权限接口
	api.RoleRouter(db, apiRouter.Group(""))

	// 用户接口
	userOption := &api.UserOption{
		Ds: db,
		R:  rClient,
	}
	api.UserRouter(userOption, apiRouter.Group(""))

	// 用户与权限映射关系的接口
	api.UserWithRoleRouter(db, apiRouter.Group(""))

	// 系统配置的接口
	middleware.AddIgnoreReadReqBodyPath("/api/sys/conf")
	api.SystemConfRouter(conf, apiRouter.Group(""))
	// 过滤掉本接口返回的数据

	addr := conf.Server.GetServerAddr()
	err = r.Run(addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
