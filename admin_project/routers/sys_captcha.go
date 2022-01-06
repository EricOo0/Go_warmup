package routers

import (
	"admin_project/core"
	"github.com/gin-gonic/gin"
)
//@author: [eric](https://github.com/EricOo0/)
//@function: Captcha
//@Tags 共有路由
//@Router /captcha [get]
//@description 请求base64编码的图像验证码
//@Success 200 {string} string "{"success":true,"id":id,"b64s":base64编码的图像}"
//@return: json
func Captcha(c *gin.Context){
	id,b64s := core.Captcha()
	c.JSON(200, gin.H{
		"success":true,
		"id":id,
		"b64s":b64s,
	})
}