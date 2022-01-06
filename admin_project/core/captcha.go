package core

import(
	"admin_project/global"

	"github.com/mojocn/base64Captcha"
)

var store = base64Captcha.DefaultMemStore

func Captcha() (string,string) {
	driver := base64Captcha.NewDriverDigit(80,240,4,0,10)
	c := base64Captcha.NewCaptcha(driver, store)

	//生成base64图像和id
	id, b64s, err := c.Generate()

	if err !=nil{
		global.GLog.Error("creat captcha failed")
		return "",""
	}
	global.GLog.Info("captcha created: "+id+" "+ b64s)
	return id,b64s
}

func Verify(CaptchaId string,Captcha string) bool{
	if CaptchaId==""|| Captcha == ""{
		return false
	}
	return store.Verify(CaptchaId, Captcha, true)
}