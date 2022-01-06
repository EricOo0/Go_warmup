package routers

import "github.com/gin-gonic/gin"

//@author: [eric](https://github.com/EricOo0/)
//@function GetinfoHandler
//@Tags 私有路由
//@Router /userinfo [get]
//@Description 获取用户信息
//@Param        data    header    string     true  "页面需要token鉴权，header带上Authorization字段"
//@Success 200 {string} string "{"success":true,"msg":"登录成功",}"
func GetinfoHandler(c *gin.Context){
	c.JSON(200, gin.H{
		"success":true,
		"msg":"登录成功",
	})
}