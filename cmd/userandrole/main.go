package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/smsapp"
	"github.com/leyle/userandrole/api"
	. "github.com/leyle/userandrole/auth"
	"github.com/leyle/userandrole/config"
	"github.com/leyle/userandrole/roleapp"
	"github.com/leyle/userandrole/userandrole"
	"github.com/leyle/userandrole/userapp"
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
	if port != "" {
		conf.Server.Port = port
	}

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

	if !conf.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := middleware.SetupGin()
	apiRouter := r.Group("/api")

	// 权限接口
	api.RoleRouter(db, apiRouter.Group(""))

	// 用户接口
	// 微信配置
	wxOpt := make(map[string]*userapp.WeChatOption)
	wxOpt[userapp.WeChatOptPlatformWeb] = &userapp.WeChatOption{
		AppId:  conf.WeChat.Web.AppId,
		Secret: conf.WeChat.Web.Secret,
		Token:  conf.WeChat.Web.Token,
		AesKey: conf.WeChat.Web.AesKey,
	}
	wxOpt[userapp.WeChatOptPlatformApp] = &userapp.WeChatOption{
		AppId:  conf.WeChat.App.AppId,
		Secret: conf.WeChat.App.Secret,
	}
	// 短信配置
	smsOpt := &smsapp.SmsOption{
		Account: conf.PhoneSms.Account,
		Passwd:  conf.PhoneSms.Password,
		Url:     conf.PhoneSms.Url,
		R:       rClient,
		Debug:   conf.PhoneSms.Debug,
		Default: true,
	}
	userOption := &api.UserOption{
		Ds: db,
		R:  rClient,
		WeChatOpt: wxOpt,
		PhoneOpt: smsOpt,
	}
	api.UserRouter(userOption, apiRouter.Group(""))

	// 用户与权限映射关系的接口
	api.UserWithRoleRouter(db, apiRouter.Group(""))

	// 系统配置的接口
	// 过滤掉本接口返回的数据
	middleware.AddIgnoreReadReqBodyPath("/api/sys/conf")
	api.SystemConfRouter(conf, apiRouter.Group(""))

	// api 文档渲染
	middleware.AddIgnoreReadReqBodyPath("/api/doc")
	r.StaticFile("/api/doc", "./doc.html")

	addr := conf.Server.GetServerAddr()
	err = r.Run(addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
