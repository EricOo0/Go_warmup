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
type DeleteUser struct{
	Username string `json:"username"`
}
type UpdateUser struct{
	Username string `json:"username"`
	Password string `json:"password"`
}
type Comment struct {
	CommentID string `gorm:"unique;not null"`
	Name string `gorm:"comment:评论者用户名"`
	Content string `gorm:"comment:评论内容"`
}

type PageInfo struct {
	Page     int `json:"page" form:"page"`         // 页码
	PageSize int `json:"pageSize" form:"pageSize"` // 每页大小
}