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
