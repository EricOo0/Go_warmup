# Day 5

使RPC服务支持HTTP协议建立连接

`通信的流程：`

* 客户端向服务器发送CONNECT请求

  ```
  CONNECT 10.0.0.1:9999/_geerpc_ HTTP/1.0
  ```

* 服务器提取出TCP连接并回复200OK

  ```
  HTTP/1.0 200 Connected to Gee RPC
  ```

* 客户端可以使用创建好的连接发送 RPC 报文，服务端处理RPC请求

  `服务端：`

  客户端发送的http请求`"/_geeprc_"`会触发`http.Handle(defaultRPCPath, server)`函数

  如果不是CONNECT则返回405错误；否则利用`w.(http.Hijacker).Hijack() `提取出这个HTTP请求的TCP连接来接管请求

  回复给客户端200 ok然后调用ServerConn等待处理rpc调用

  ```go
  const (
  	Connected        = "200 Connected to Gee RPC"
  	DefaultRPCPath   = "/_geeprc_"
  	DefaultDebugPath = "/debug/geerpc"
  )
  // server 实现了ServeHTTP函数，即实现了handler接口，http请求来了就会调用
  // ServeHTTP implements an http.Handler that answers RPC requests.
  func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  	if req.Method != "CONNECT" {
  		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
  		w.WriteHeader(http.StatusMethodNotAllowed)
  		_, _ = io.WriteString(w, "405 must CONNECT\n")
  		return
  	}
  	conn, _, err := w.(http.Hijacker).Hijack()
  	if err != nil {
  		log.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
  		return
  	}
  	_, _ = io.WriteString(conn, "HTTP/1.0 "+Connected+"\n\n")
  	s.ServerConn(conn)
  }
  
  // HandleHTTP registers an HTTP handler for RPC messages on rpcPath.
  // It is still necessary to invoke http.Serve(), typically in a go statement.
  func (s *Server) HandleHTTP() {
  	http.Handle(DefaultRPCPath, s)
  }
  
  // 设置默认handler方便测试
  func HandleHTTP() {
  	DefaultServer.HandleHTTP()
  }
  ```

​	`客户端测`

​	服务端侧已经可以接受Conncect请求了，客户端需要做的就是发起connect

​	调用NewHttpClient使用HTTP创建一个连接

```go
// NewHTTPClient new a Client instance via HTTP as transport protocol
func NewHTTPClient(conn net.Conn, opt *service.Option) (*Client, error) {
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", service.DefaultRPCPath)) //发出HTTP请求

	// Require successful HTTP response
	// before switching to RPC protocol.
  //服务器的回复不是http格式，要转换成http格式的回复
  //ReadResponse reads and returns an HTTP response from r
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == service.Connected {
		return NewClient(conn, opt)//建立一个tcp客户端
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	return nil, err
}

// DialHTTP connects to an HTTP RPC server at the specified network address
// listening on the default HTTP RPC path.
func DialHTTP(network, address string, opts ...*service.Option) (*Client, error) {
	return dialTimeout(NewHTTPClient, network, address, opts...)
}

```

简化不同协议的可以统一调用入口：

```go
func XDial(rpcAddr string, opts ...*Option) (*Client, error) {
	parts := strings.Split(rpcAddr, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("rpc client err: wrong format '%s', expect protocol@addr", rpcAddr)
	}
	protocol, addr := parts[0], parts[1]
	switch protocol {
	case "http":
		return DialHTTP("tcp", addr, opts...)
	default:
		// tcp, unix or other transport protocol
		return Dial(protocol, addr, opts...)
	}
}
```

测试demo：

```go
package main

import (
	"context"
	"geerpc/client"
	"geerpc/service"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

//服务端测
type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addrCh chan string) {
	var foo Foo
	l, _ := net.Listen("tcp", ":9999")
	_ = service.Register(&foo)
	service.HandleHTTP() //处理http
	addrCh <- l.Addr().String()
	_ = http.Serve(l, nil) //相当于listenandserver函数，监听这个接口的http请求
}

//客户端测
func call(addrCh chan string) {
	client, _ := client.DialHTTP("tcp", <-addrCh) //使用http调用
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call(context.Background(), "Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)
	ch := make(chan string)
	go call(ch)
	startServer(ch)
}
```

使用HTTP协议(CONNECT方法)和服务端建立了连接，之后的客户端的RPC请求应该就是普通的tcp包了