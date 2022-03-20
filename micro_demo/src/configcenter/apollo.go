package main

import (
	"fmt"
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
)

func main() {
	c := &config.AppConfig{
		AppID:          "testApplication_yang",
		Cluster:        "dev",
		IP:             "http://106.54.227.205:8080",
		NamespaceName:  "dubbo3",
		IsBackupConfig: true,
		Secret:         "6ce3ff7e96a24335a9634fe9abca6d51",
	}

	client, _ := agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return c, nil
	})
	fmt.Println("初始化Apollo配置成功")

	//Use your apollo key to test
	cache := client.GetConfigCache(c.NamespaceName)
	value, _ := cache.Get("key")
	fmt.Println(value)
}
