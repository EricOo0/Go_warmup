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

鉴权：jwt

图像验证码生成 base64captcha  
加密 md5  

已有接口：

公共：

​	/register：[post] 注册用户并存储到数据库，后台会先验证前端给过来的用户名密码是否符合要求且验证用户是否存在

​	/captcha：[get] 获取图形验证码的api，后台会给前端返回base64编码后的图形验证码

​	/login：[post] 用户登录功能，后台先判断验证码是否正确，然后判断账户密码是否正确，成功返回success和一个jwt生成的token，用于身份鉴权

私有：

​	/getuserinfo [get]. 获取用户信息

​	/deleteuser	[post] 删除指定用户信息，需判断用户身份和权限

​	/changepassword [post] 修改密码



​	/getcomment [get] 获取评论列表 需要带上pagesize和page参数

​	/addcomment [post] 添加评论

​	/deletecomment[post] 删除评论

