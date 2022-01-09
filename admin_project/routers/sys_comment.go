package routers

import (
	"admin_project/global"
	"admin_project/sysRequest"
	"admin_project/util"
	"github.com/gin-gonic/gin"
)
//@function AddComment
//@author: [eric](https://github.com/EricOo0/)
//@Tags 私有路由
//@description 增加评论
//@Param        data    body    global.Comment     true  "含用户名和内容即可"
//@Success 200 {string} string "{"success":true,"msg":"添加评论成功","err":""}"
//@Router /addcomment [post]
//@return json
func AddComment(c *gin.Context){
	var comment global.Comment
	_ = c.ShouldBindJSON(&comment)
	username,_ := util.GetPriviledge(c)
	comment.Name=username
	err := global.G_DB.Table("comments").Create(&comment).Error
	if err != nil{
		c.JSON(400,gin.H{
			"success":false,
			"error":err,
			"msg":"添加评论失败",
		})
		return
	}
	c.JSON(200,gin.H{
		"success":true,
		"error":"",
		"msg":"添加评论成功",
	})
}
//@function DeleteComment
//@author: [eric](https://github.com/EricOo0/)
//@Tags 私有路由
//@description 删除评论
//@Param        data    body    global.Comment     true  "有commentid即可"
//@Success 200 {string} string "{"success":true,"msg":"删除成功","err":""}"
//@Router /deletecomment [post]
//@return json
func DeleteComment(c *gin.Context){
	var comment global.Comment
	_ = c.ShouldBindJSON(&comment)
	//username,_ := util.GetPriviledge(c)
	err := global.G_DB.Table("comments").Where("comment_id = ?",comment.CommentID).Delete(&comment).Error
	if err != nil{
		c.JSON(400,gin.H{
			"success":false,
			"error":err,
			"msg":"删除失败",
		})
		return
	}
	c.JSON(200,gin.H{
		"success":true,
		"error":"",
		"msg":"删除成功",
	})
}
//@function GetComment
//@author: [eric](https://github.com/EricOo0/)
//@Tags 私有路由
//@description 增加评论
//@Param        data    body    sysRequest.PageInfo     true  "一页的评论数和页数"
//@Success 200 {string} string "{"success":true,"commentlist":{{},{}},"msg":"添加评论成功","err":""}"
//@Router /getcomment [get]
//@return json
func GetComment(c *gin.Context){
	//一页50条评论
	var comments sysRequest.PageInfo
	var commentlist []global.Comment
	_ = c.ShouldBindJSON(&comments)
	limits := comments.PageSize
	offset := (comments.Page-1)*comments.PageSize
	err := global.G_DB.Table("comments").Limit(limits).Offset(offset).Find(&commentlist).Error
	if err != nil{
		c.JSON(400,gin.H{
			"success":false,
			"error":"err",
			"msg":"获取评论失败",
		})
		return
	}
	c.JSON(200,gin.H{
		"success":true,
		"error":"err",
		"msg":"获取评论成功",
		"comments":commentlist,
	})
	return
}

