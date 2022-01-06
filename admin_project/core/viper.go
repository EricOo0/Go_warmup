package core

import (
	"admin_project/global"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)
func Viper() *viper.Viper{
	config := viper.New()
	config.SetConfigName("config")
	config.AddConfigPath("./")
	//设置配置文件类型
	config.SetConfigType("yaml")
	//读取配置
	err := config.ReadInConfig()

	if err != nil {
		global.GLog.Error("Fatal error config file:", zap.Error(err))
		panic(fmt.Errorf("Fatal error config file: %s \n", err))

	}

	config.WatchConfig()
	config.OnConfigChange(func(e fsnotify.Event){
		err := config.ReadInConfig()
		if err != nil {
			global.GLog.Error("Fatal error config file:", zap.Error(err))
			panic(fmt.Errorf("Fatal error config file: %s \n", err))

		}
		if err := config.Unmarshal(&global.G_Config); err != nil{
			global.GLog.Error("unable to unmarshal config:", zap.Error(err))
		}
	})
	if err := config.Unmarshal(&global.G_Config); err != nil{
		global.GLog.Error("unable to unmarshal config:", zap.Error(err))
	}//加载配置
	return config
}