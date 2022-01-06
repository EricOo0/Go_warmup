package core
import(
	"admin_project/global"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Db()(db *gorm.DB){
	m := global.G_Config.Mysql
	mysqlConfig:=mysql.Config{
		DSN:m.Dsn(),
		//DSN:"weizhifeng:weizhifeng10@tcp(127.0.0.1:3306)/adminDB?charset=utf8mb4&parseTime=True&loc=Local",
	}
	db,_ = gorm.Open(mysql.New(mysqlConfig),&gorm.Config{})
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(m.Max_idle_conns)
	sqlDB.SetMaxOpenConns(m.Max_open_conns)
	return db

}