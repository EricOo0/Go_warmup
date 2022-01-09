package main

import (
	"admin_project/core"
	_ "admin_project/docs"
	"admin_project/global"
	"admin_project/middlerware"
	"admin_project/routers"
	"fmt"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @termsOfService http://swagger.io/terms/
func main(){
	//启动日志
	global.GLog = core.Zap()
	global.GLog.Debug("server runing")
	//启动配置读取
	global.G_Viper = core.Viper()
	//连接数据库
	global.G_DB = core.Db()
	global.G_DB.AutoMigrate(&global.User{},&global.Comment{})
	db,_ := global.G_DB.DB()
	defer db.Close()
	//u := User{Password: "test",Username: "test4"}
	//gDb.Create(&u)


	//admin 服务器启动
	s := gin.Default()
	//启动接口文档swagger
	s.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	fmt.Println("在线api文档部署在：http://localhost:8080/swagger/index.html")
	//公共路由 注册，登录，验证码
	s.GET("/captcha", routers.Captcha)
	s.POST("/register",routers.RegisterHandler)
	s.POST("/login", routers.LoginHandler)
	//用户路由   访问前需要认证token
	usrRouter := s.Group("")

	usrRouter.Use(middlerware.Auth)
	{
		usrRouter.GET("userinfo", routers.GetinfoHandler)
		usrRouter.POST("deleteUser", routers.DeleteUserHandler)
		usrRouter.POST("changepassword", routers.ChangePassword)


		s.POST("/addcomment", routers.AddComment)
		s.POST("/deletecomment", routers.DeleteComment)
		s.GET("/getcomment", routers.GetComment)
	}

	// 服务启动
	if err := s.Run(); err != nil {
		global.GLog.Error("server is fail!")
	}
}

// ShowAccount godoc
// @Summary Show a account
// @Tags Example API
// @Description get string by ID
// @Produce  json
// @Success 200
// @Router /ping [get]
func pang(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

