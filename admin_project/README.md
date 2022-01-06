# admin系统Demo
1、 修改config.yaml中连接数据库的配置，主要是username和password和dbname
配置且连接成功后会自动去建表  
2、go mod tidy 安装一些依赖包  
3、go build -o sever admin_project 编译  
4、./sever  
5、运行成功后：  
提供给前端使用和测试的在线api文档部署在：http://localhost:8080/swagger/index.html  
日志文件打印在LOG文件夹下

用到的框架：  
日志框架 zap  
日志切割 lamberjack  
数据库 gorm  
配置管理 viper  
web后台 gin  
api文档生成 swagger  
图像验证码生成 base64captcha  
加密 md5  