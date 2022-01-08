package routers

import (
	"admin_project/global"
	"admin_project/sysRequest"
	"admin_project/util"
	"fmt"
	"github.com/gin-gonic/gin"
)

//@author: [eric](https://github.com/EricOo0/)
//@function GetinfoHandler
//@Tags 私有路由
//@Router /userinfo [get]
//@Description 获取用户信息
//@Param        data    header    string     true  "页面需要token鉴权，header带上Authorization字段"
//@Success 200 {string} string "{"success":true,"msg":"hello:user",}"
func GetinfoHandler(c *gin.Context){
	username,_ := util.GetPriviledge(c)
	msg := fmt.Sprintf("hello: {%s}",username)
	c.JSON(200, gin.H{
		"success":true,
		"msg":msg,
	})
}
//@author: [eric](https://github.com/EricOo0/)
//@function DeleteUserHandler
//@Tags 私有路由
//@Router /deleteUser [post]
//@Description 获取用户信息
//@Param        data    header    string     true  "页面需要token鉴权，header带上Authorization字段"
//@Param        username    body    string     true  "要删除的username"
//@Success 200 {string} string "{"success":true,"msg":"删除成功",err:"error reson",}"
func DeleteUserHandler(c *gin.Context){

	var deleteUser sysRequest.DeleteUser
	_ = c.ShouldBindJSON(&deleteUser)
	_,priv := util.GetPriviledge(c)
	if priv != global.Priv_Admin {
		c.JSON(400, gin.H{
			"success":false,
			"msg":"权限不足",

		})
	}else{
		var u global.User
		err := global.G_DB.Table("users").Where("username = ?",deleteUser.Username).Delete(&u).Error
		c.JSON(200, gin.H{
			"success":true,
			"msg":"删除成功",
			"error":err,

		})
	}

}
//@author: [eric](https://github.com/EricOo0/)
//@function ChangePassword
//@Tags 私有路由
//@Router /changepassword [post]
//@Description 修改用户密码
//@Param        data    header    string     true  "页面需要token鉴权，header带上Authorization字段"
//@Param        username    body    string     true  "要修改的username"
//@Param        password    body    string     true  "新的密码"
//@Success 200 {string} string "{"success":true,"msg":"修改成功",err:"error reson",}"
func ChangePassword(c *gin.Context){

	var u sysRequest.UpdateUser
	_ = c.ShouldBindJSON(&u)
	username,priv := util.GetPriviledge(c)
	if username != u.Username{
		//修改其他用户密码
		if priv != global.Priv_Admin{
			c.JSON(400, gin.H{
				"success":false,
				"msg":"权限不足",
				"error":"只能修改自己的密码",

			})
			return
		}
	}
	//md5 加密
	password :=util.Md5([]byte(u.Password))
	fmt.Println(u)
	err := global.G_DB.Model(global.User{}).Where("username = ?",u.Username).Update("password",password).Error
	if err != nil{
		c.JSON(400, gin.H{
			"success":false,
			"msg":"修改失败",
			"error":err,

		})
		return
	}
	c.JSON(400, gin.H{
		"success":true,
		"msg":"修改成功",
		"error":err,

	})
}