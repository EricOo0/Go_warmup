package core

import (
	"admin_project/global"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

func Db() *gorm.DB {

	m := global.G_Config.Mysql
	fmt.Println(m)
	fmt.Println(m.Dsn())
	mysqlConfig:=mysql.Config{
		DSN:m.Dsn(),

		//DSN:"weizhifeng:weizhifeng10@tcp(127.0.0.1:3306)/adminDB?charset=utf8mb4&parseTime=True&loc=Local",
	}
	logfile,_ := os.Create("./Log/sql.log")
	newLogger := logger.New(
		log.New(logfile, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,   // 慢 SQL 阈值
			LogLevel:      logger.Info, // Log level
			Colorful:      true,         // 禁用彩色打印
		},
	)
	db,_ := gorm.Open(mysql.New(mysqlConfig),&gorm.Config{
		Logger : newLogger,
	})

	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(m.MaxIdleConns)
	sqlDB.SetMaxOpenConns(m.MaxOpenConns)

	return db

}