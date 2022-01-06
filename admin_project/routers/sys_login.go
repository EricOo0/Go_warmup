package routers

import (
	"admin_project/core"
	"admin_project/global"
	"admin_project/middlerware"
	"admin_project/sysRequest"
	"crypto/md5"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

//@function LoginHandler
//@author: [eric](https://github.com/EricOo0/)
//@Tags 共有路由
//@description 提交登录信息
//@Param        data    body    sysRequest.Login     true  "上传登录信息和验证码"
//@Success 200 {string} string "{"success":true,"msg":"登录成功","token":"aaa.bbb.ccc"}"
//@Router /login [post]
//@return json
func LoginHandler(c *gin.Context){
	var l sysRequest.Login
	_ = c.ShouldBindJSON(&l)
	id := l.Id
	b64s :=l.B64s
//	id:=c.PostForm("id")
//	b64s:=c.PostForm("b64s")
	if match:=core.Verify(id,b64s);!match{
		c.JSON(401,gin.H{
			"success":false,
			"msg":"验证码错误",
			"token":"",
		})
		return
	}
	//username := c.PostForm("username")
	//password := c.PostForm("password")
	username :=l.Username
	password :=l.Password
	var user global.User
	err := global.G_DB.Where("Username = ? And Password = ?",username,password).First(&user).Error
	if err !=nil{
		c.JSON(404,gin.H{
			"success":false,
			"msg":"用户名或密码错误",
			"token":"",
		})
		return
	}
	//登录成功，返回一个token作为鉴权
	token,_ :=middlerware.CreatToken(&user)
	c.JSON(200, gin.H{
		"success":true,
		"msg":"登录成功",
		"token":token,
	})
}
//@author: [eric](https://github.com/EricOo0/)
//@Tags 共有路由
//@function RegisterHandler
// @Router /register [post]
//@description 提交注册用户信息
//@Param        data  body  sysRequest.Register       true  "注册用户账户,密码"
// @Success 200 {string} string "{"success":true,"msg":"注册成功"}"
//@Produce  application/json
func RegisterHandler(c *gin.Context){
	var l sysRequest.Register
	_ = c.ShouldBindJSON(&l)
//	username := c.PostForm("username")
//	password := c.PostForm("password")
	if len(l.Username)<6 || len(l.Username)>15{
		c.JSON(400, gin.H{
			"success":false ,
			"msg":"用户名不符合要求，长度应在6-15之间",
		})
		return
	}
	if l.Password=="" {
		c.JSON(400, gin.H{
			"success":false ,
			"msg":"密码不能为空",
		})
		return
	}
	str :=[]byte(l.Password)
	w :=md5.New()//初始化一个MD5对象
	w.Write(str) //str为要加密的字符串
	tmp := w.Sum(nil) //计算校验和
	pwd := hex.EncodeToString(tmp)
	u := &global.User{Username: l.Username,Password: pwd}
	err := global.G_DB.Create(&u).Error
	if err !=nil{
		c.JSON(400, gin.H{
			"success":false,
			"msg":err,
		})
		return
	}
	global.GLog.Info("user register:",zap.String("user",u.Username))
	c.JSON(200, gin.H{
		"success":true,
		"msg":"注册成功",
	})

}