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

	/*
		reg := consul.NewRegistry(registry.Addrs("127.0.0.1:8500"))
	*/
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
