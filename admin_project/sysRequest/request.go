package sysRequest
type Login struct{
	Username string `json:"username"`
	Password string `json:"password"`
	Id string `json:"id"`
	B64s string `json:"b64s"`
}

type Register struct{
	Username string `json:"username"`
	Password string `json:"password"`
}