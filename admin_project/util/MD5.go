package util

import (
	"admin_project/global"
	"admin_project/middlerware"
	"crypto/md5"
	"encoding/hex"
	"github.com/gin-gonic/gin"
)

func Md5(str []byte) string{
	//md5 加密

	w :=md5.New()//初始化一个MD5对象
	w.Write(str) //str为要加密的字符串
	tmp := w.Sum(nil) //计算校验和
	password := hex.EncodeToString(tmp)
	return password
}
func GetPriviledge(c *gin.Context) (string,int){
	tokenString := c.GetHeader("Authorization")
	username,_ := middlerware.PhaseToken(tokenString)
	type user_priv struct {
		Priv	int `gorm:"default:2" json:"privilege" `
	}
	var priv user_priv
	global.G_DB.Table("users").Select("Priv").Where("username = ?",username).Scan(&priv);
	return username,priv.Priv
}