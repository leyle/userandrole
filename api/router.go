package api

import (
	"github.com/gin-gonic/gin"
	"github.com/leyle/ginbase/dbandmq"
)

func RoleRouter(db *dbandmq.Ds, g *gin.RouterGroup) {
	auth := g.Group("", func(c *gin.Context) {
		Auth(c)
	})

	roleR := auth.Group("/role")

	itemR := roleR.Group("/item")
	{
		// 新建 item
		itemR.POST("", func(c *gin.Context) {
			CreateItemHandler(c, db)
		})

		// 修改 item
		itemR.PUT("/:id", func(c *gin.Context) {
			UpdateItemHandler(c, db)
		})

		// 删除 item
		itemR.DELETE("/:id", func(c *gin.Context) {
			DeleteItemHandler(c, db)
		})

		// 读取 item 明细
		itemR.GET("/:id", func(c *gin.Context) {
			GetItemInfoHandler(c, db)
		})

		// 搜索 item
		roleR.GET("/items", func(c *gin.Context) {
			QueryItemHandler(c, db)
		})
	}
}

func UserRouter(uo *UserOption, g *gin.RouterGroup) {
	auth := g.Group("", func(c *gin.Context) {
		Auth(c)
	})

	userR := auth.Group("/user")
	{
		// 新建一个账户密码，只能通过拥有相关权限的人来调用
		userR.POST("/idpasswd", func(c *gin.Context) {
			CreateLoginIdPasswdAccountHandler(c, uo)
		})

		// 修改密码,修改自己的密码
		userR.POST("/idpasswd/changepasswd", func(c *gin.Context) {
			UpdatePasswdHandler(c, uo)
		})
	}

	// 不需要 auth 的
	noAuthR := g.Group("/user")
	{
		noAuthR.POST("/idpasswd/login", func(c *gin.Context) {
			LoginByIdPasswdHandler(c, uo)
		})
	}
}