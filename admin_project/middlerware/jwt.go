package middlerware

import (
	"admin_project/global"
	"github.com/dgrijalva/jwt-go"
	"time"
)
type JWT struct {
	SigningKey []byte
}
//jwt的payload部分
type Claims struct {
	Username string
	jwt.StandardClaims
}

func NewJWT() *JWT{
	return &JWT{
		SigningKey:  []byte("weizhifeng"),
	}
}
//颁发JWT token
func CreatToken(u *global.User) (string,error){
	jwtkey:= NewJWT()
	claim := Claims{
		Username: u.Username,
		StandardClaims:jwt.StandardClaims{
			IssuedAt: time.Now().Unix(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Minute).Unix(),//过期时间7*24分钟
			Issuer: "weizhifeng",
			Subject:   "user token", //签名主题

		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)

	tokenString, err := token.SignedString(jwtkey.SigningKey)//签名加密

	return tokenString,err
}
func PhaseToken(tokenString string) (string,error){
	claim := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claim, func(token *jwt.Token) (i interface{}, err error) {
		return  []byte("weizhifeng"), nil
	})
	if err!=nil || !token.Valid{  // 判断Tocken是否有效

		return "",err
	}
	return claim.Username,err
}
