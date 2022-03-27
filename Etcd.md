# Etcd

etcd 是一个高可用强一致性的键值仓库，基于raft协议，可用于共享配置，服务注册，服务发现的registry

安装：

​	brew install etcd

​	brew service start etcd 启动  127.0.0.1:2379

安装webui:

​	 git clone https://github.com/henszey/etcd-browser.git

golang 中使用

​	 "github.com/coreos/etcd/clientv3"

# RabbitMQ

安装 ：

​	brew install rabbitmq

启动：

​	rabbitmq-server 

​	http://localhost:15672 默认的用户名密码都是guest。 代码连接要连5672

​	rabbitmq-plugins enable rabbitmq_management

​	rabbitmqctl stop 关闭

go get github.com/streadway/amqp



# kafka

kafka 不兼容m1 使用conluent

https://www.cnblogs.com/cxl-/p/14732902.html

# protobuf



使用micro注册服务到etcd

安装protoc:

​	go get github.com/micro/micro

​	go get github.com/micro/protoc-gen-micro/v2

​	go get -u github.com/golang/protobuf/protoc-gen-go

安装成功后用protoc生成代码：

​	 protoc --proto_path=./proto/pb/:. --micro_out=. --go_out=. ./proto/pb/greeter.proto



# micro v2

服务注册:注册前需要使用protobuf转换api成go代码

proto

```protobuf
syntax = "proto3";
package api;
option go_package = "./proto;api";
service Greeter{
  rpc Hello (HelloRequest) returns (HelloResponse){}
  rpc Bye (ByeRequest) returns (ByeResponse){}
}
message HelloRequest{
  string name = 1;
}
message HelloResponse{
  string greeting = 1;
}
message ByeRequest{
  string name =1;
}
message ByeResponse{
  string bye=1;
}
```

server.go

```go
package main

import (
   "context"
   "fmt"
   "github.com/micro/go-micro/v2"
   "github.com/micro/go-micro/v2/broker"
   "github.com/micro/go-micro/v2/registry"
   "github.com/micro/go-micro/v2/registry/etcd"
   "github.com/micro/go-plugins/broker/rabbitmq/v2"
   api "micro_demo/proto"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *api.HelloRequest, resp *api.HelloResponse) error {
   fmt.Println("Hello:", req.Name, "welcome to micro demo")
   resp.Greeting = "wish you a good day"
   return nil
}
func (g *Greeter) Bye(ctx context.Context, req *api.ByeRequest, resp *api.ByeResponse) error {
   fmt.Println("Hello:", req.Name, "welcome to micro demo")
   resp.Bye = "good bye ! wish you a good day"
   return nil
}
func main() {
   //配置etcd作为配置中心，配置路径为127.0.0.1:2379
   reg := etcd.NewRegistry(
      registry.Addrs("127.0.0.1:2379"),
   )
   //消息队列使用rabbitmq
   brk := rabbitmq.NewBroker(
      broker.Addrs("amqp://guest:guest@localhost:5672"),
   )

   srv := micro.NewService(
      micro.Name("micro_test"),
      micro.Registry(reg),
      micro.Broker(brk),
   )
   srv.Init()
   //把服务注册到etcd中去
   api.RegisterGreeterHandler(srv.Server(), &Greeter{})
   srv.Run()

}
```

Server 把greeter实例通过`api.RegisterGreeterHandler(srv.Server(), &Greeter{})`注册到etcd中去，服务名为`micro_test`

客户端：

client.go

```go
package main

import (
   "context"
   "fmt"
   "github.com/micro/go-micro/v2"
   "github.com/micro/go-micro/v2/registry"
   "github.com/micro/go-micro/v2/registry/etcd"
   api "micro_demo/proto"
)

func main() {
   reg := etcd.NewRegistry(
      registry.Addrs("127.0.0.1:2379"),
   )
   srv := micro.NewService(
      micro.Name("micro_test_client"),
      micro.Registry(reg),
   )
   srv.Init()
   //把服务注册到etcd中去
   cli := api.NewGreeterService("micro_test", srv.Client())
   rsp, err := cli.Hello(context.TODO(), &api.HelloRequest{Name: "client"})
   if err != nil {
      fmt.Println("err:", err)
   }
   fmt.Println("rsp,", rsp)
}
```

客户端通过`cli := api.NewGreeterService("micro_test", srv.Client()`得到一个greeter对象，调Hello的过程中会调用`call`，使用micro的selector去etcd寻找服务提供者的ip

同一个服务名注册多次会有多个实例：

![image-20220313155324206](image/image-20220313155324206.png)

# jaeger

链路追踪框架

https://www.bilibili.com/video/BV1J3411C7Sz?from=search&seid=18342260911654687631&spm_id_from=333.337.0.0

https://blog.csdn.net/liyunlong41/article/details/87932953





# apollo配置管理

可以从可视化的ui上批量配置进程的配置

```go
package main

import (
   "fmt"
   "github.com/apolloconfig/agollo/v4"
   "github.com/apolloconfig/agollo/v4/env/config"
)

func main() {
   c := &config.AppConfig{
      AppID:         "test",
      Cluster:       "test-cluster",
      IP:            "http://localhost:8080",
      NamespaceName: "test1",
   }

   client, _ := agollo.StartWithConfig(func() (*config.AppConfig, error) {
      return c, nil
   })

   fmt.Println("初始化Apollo配置成功")
   value := client.GetConfig(c.NamespaceName).GetValue("key")
   //value, _ := cache.Get("key")
   fmt.Println(value)

}
```

# cron 定时任务框架

“github/robfig/cron”

精确到秒的定时任务框架

------

# 调度平台

xxl_job

