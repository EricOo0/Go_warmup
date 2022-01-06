package global

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)
var(
	GLog *zap.Logger
	G_DB *gorm.DB
	G_Viper *viper.Viper
	G_Config Config
)
type User struct{
	ID        uint `gorm:"primarykey"`
	Username string `gorm:"not null;unique;comment:用户账户" json:"username"`
	Password string `gorm:"comment:用户登录名" json:"password"`
	CreatedAt       time.Time `json:"creattime"`
	UpdatedAt       time.Time `json:"updatetime"`
}
type Config struct{
	Mysql Mysql
}
type Mysql struct {
	Path string `yaml:"path"`
	Port string	`yaml:"port"`
	Config string `yaml:"config"`
	Db_name string `yaml:"db-name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Max_idle_conns int `yaml:"max-idle-conns"`
	Max_open_conns int 	`yaml:"max-open-conns"`
	Log_mode int	`yaml:"log-mode"`
	Log_zap bool	`yaml:"log-zap"`

}
func (m *Mysql) Dsn() string{
	return m.Username + ":" + m.Password + "@tcp(" + m.Path + ":" + m.Port + ")/" + m.Db_name + "?" + m.Config
}