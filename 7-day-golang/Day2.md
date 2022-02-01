# Day2

今天主要是实现一个实现一个支持异步和并发的高性能客户端

------

一个支持rpc调用的函数大概长下面这个样子

```go
func (t *T) MethodName(argType T1, replyType *T2) error
```

因此我们先抽象出一个调用结构体Call，包含了一次rpc调用包含的所有信息，每个call还有个done函数用来通知调用方调用已结束

client/client.go

```go
type Call struct{
   Seq uint64 //序列号，标志一次请求
   ServiceMethod string //请求服务和方法
   Args interface{} //调用参数
   Reply interface{} //回复参数
   Error error //错误信息
   Done chan *Call //当一次调用完成，用于通知调用方

}
//支持异步调用，使用channel来通知调用方
func (call *Call) done() {
	call.Done <- call
}
```

rpc客户端的重要部分是client结构，每个client可以发出多次调用，所以他需要包含下列字段：

- Codec 是消息的编解码器，和服务端类似，用来序列化将要发送出去的请求，以及反序列化接收到的响应。
- sending 是一个互斥锁，和服务端类似，为了保证请求的有序发送，即防止出现多个请求报文混淆。
- header 是每个请求的消息头，header 只有在请求发送时才需要，而请求发送是互斥的，因此每个客户端只需要一个，声明在 Client 结构体中可以复用。
- seq 用于给发送的请求编号，每个请求拥有唯一编号。
- pending 是一个存储调用的map，存储未处理完的请求，键是编号，值是 Call 实例。
- Closed 和 shutdown 任意一个值置为 true，则表示 Client 处于不可用的状态，但有些许的差别，closed 是用户主动关闭的，即调用 `Close` 方法，而 shutdown 置为 true 一般是有错误发生。 

```go
type Client struct{
   c codec.Codec
   h codec.Header
   seq uint64
   opt *geerpc.Option
   sending sync.Mutex
   mu sync.Mutex
   pending map[uint64]*Call
   closed bool  // user has called Close
   shutdown bool // server has told us to stop
}

var _ io.Closer = (*Client)(nil) //这一步是为了保证client继承了closer接口
var ErrShutdown = errors.New("connection is shut down")
func (cli *Client) Close() error {
	cli.mu.Lock()
	defer  cli.mu.Unlock()
	if cli.closed {
		return ErrShutdown
	}
	cli.closed = true
	return cli.c.Close()
}

// IsAvailable return true if the client does work 判断客户端是否可用
func (cli *Client) IsAvailable() bool {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	return !cli.shutdown && !cli.closed
}
```

client 结构内部需要有注册调用，删除调用，终止调用三个内部方法

* registerCall 把一个调用Call注册到client中pending结构中，client的seq+1
* removeCall 把一个调用Call从client中删除，可能是调用已结束或者出错
* terminateCalls client出错，把所有的Call都结束掉

```go
//把这个调用注册进clien实例；注册成功返回序列号
func (cli *Client) registerCall(call *Call)  (uint64,error){
   cli.mu.Lock()
   defer cli.mu.Unlock()
   if cli.shutdown || cli.closed{
      return 0,ErrShutdown
   }
   //如果客户端没有关闭
   call.Seq=cli.seq
   cli.pending[call.Seq]=call
   cli.seq++
   return call.Seq,nil
}
func (cli *Client) removeCall(seq uint64) *Call {
   cli.mu.Lock()
   defer cli.mu.Unlock()
   call := cli.pending[seq]
   delete(cli.pending,seq)
   return call
}
func (cli *Client) terminateCalls(err error){
   cli.sending.Lock()
   defer cli.sending.Unlock()
   //先停止发送，再关闭
   cli.mu.Lock()
   defer cli.mu.Unlock()
   cli.shutdown = true
   for _,call := range cli.pending{
      call.Error=err
      call.done()
   }

}
```

有了上面的三个基本函数，就可以实现客户端client最基本的功能了，发送请求和接收回复

接受回复有可能有下面几种情况

* call调用不存在，即这个call可能已经被client结束了，那么把回复body接收完继续接受他的call，不管这个了
* call存在，但是接收到的header有错误，即服务端回复了错误，那么结束这个call并把错误信息写入call结构体
* 正常情况，接收回复信息到reply结构中，结束这个call调用

发送请求的流程是：

* 注册Call调用到client中
* 修改cli的header的信息用于发送这个调用
* 编码并且发送请求

```go
//对一个客户端端来说，接收响应、发送请求是最重要的 2 个功能。
//接受请求
func (cli *Client) receive(){
   var err error
   for err == nil{
      var h codec.Header
      if err = cli.c.ReadHeader(&h);err !=nil{
         break
      }
      call := cli.removeCall(h.Seq)
      switch  {
      case call == nil:
         //call位nil，证明这个调用已经被停止了
         err = cli.c.ReadBody(nil) //把body从io读出来
      case h.Error !="":
         //call存在但是error不为空，服务端报错
         call.Error = fmt.Errorf(h.Error)
         err = cli.c.ReadBody(nil)
         call.done()
      default:
         err = cli.c.ReadBody(call.Reply)
         if err != nil {
            call.Error = errors.New("reading body " + err.Error())
         }
         call.done()
      }
   }
   //call 有错误，要结束这个客户端
   cli.terminateCalls(err)
}
// 发送请求
func (cli *Client) send(call *Call) {
	cli.sending.Lock()
	defer cli.sending.Unlock()
	seq, err := cli.registerCall(call)//发送得先注册到client
	if err != nil {
		call.Error = err
		call.done()
		return
	}
	cli.h.ServiceMethod = call.ServiceMethod
	cli.h.Seq = seq
	cli.h.Error=""

	//encode
	if err := cli.c.Write(&cli.h,&call.Args);err!=nil{
		call := cli.removeCall(seq)
		if call !=nil{
			call.Error = err
			call.done()
		}
	}

}

```

现在客户端已经具有发送和接收的能力，接下来我们写一下客户端client的构造和初始化函数：

首先new一个client实例出来然后将option发送给server，协商好通信格式

```go
//创建Client实例
func NewClient(conn net.Conn,opt *geerpc.Option) (*Client,error){
   f := codec.NewCodecFuncMap[opt.CodecType]
   if f ==nil{
      err := fmt.Errorf("invalid codec type %s", opt.CodecType)
      log.Println("rpc client: codec error:", err)
      return nil, err
   }
   //要把option先发给服务端
   if err := json.NewEncoder(conn).Encode(opt); err != nil{
      log.Println("rpc client: options error: ", err)
      _ = conn.Close()
      return nil, err
   }
   return newClientCodec(f(conn), opt), nil
}
func newClientCodec(c codec.Codec,opt *geerpc.Option) *Client{
   client := &Client{
      c:c,
      seq:1,
      opt:opt,
      pending: make(map[uint64]*Call),
   }
   go client.receive()
   return client
}
```

客户端暴露出来的函数主要有以下三个：

* Dial 函数：连接server并且创建出一个client实例
* Call函数：发出一个RPC调用，回去掉Go执行一个调用；`Call` 是对 `Go` 的封装，阻塞 call.Done，等待响应返回，是一个同步接口
* Go：执行一个调用

```go

//发送主要是实现一个Dial函数，调用远端函数
func Dial( network,address string,opts ...*geerpc.Option) (client *Client,err error){
   opt, err := parseOptions(opts...)
   if err != nil {
      return nil, err
   }
   conn, err := net.Dial(network, address) //连接服务器
   if err != nil {
      return nil, err
   }
   defer func() {
      if client == nil {
         _ = conn.Close()
      }
   }()
   //创建客户端
   client,e := NewClient(conn,opt)
   return client,e
}
//用于解析用户的option要求
func parseOptions(opts ...*geerpc.Option)(*geerpc.Option,error){

   if len(opts) == 0 || opts[0] == nil {
      fmt.Println(geerpc.DefaultOption)
      return geerpc.DefaultOption, nil
   }
   if len(opts) != 1 {
      return nil, errors.New("number of options is more than 1")
   }
   opt := opts[0]
   opt.MagicNumber = geerpc.DefaultOption.MagicNumber
   if opt.CodecType == "" {
      opt.CodecType =geerpc.DefaultOption.CodecType
   }

   return opt, nil
}


func (cli *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	cli.send(call)
	return call
}

func (cli *Client) Call(serviceMethod string, args, reply interface{}) error {
	call := <-cli.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}
```

------

接下来修改一下main函数，测试我们的client：

```go
package main

import (
   "fmt"
   "geerpc"
   "geerpc/client"
   "log"
   "net"
   "sync"
   "time"
)

func startServer(addr chan string) {
   // pick a free port
   l, err := net.Listen("tcp", ":0")
   if err != nil {
      log.Fatal("network error:", err)
   }
   log.Println("start rpc server on", l.Addr())
   addr <- l.Addr().String()
   geerpc.Accept(l)
}

func main() {
   addr := make(chan string) //addr是一个 string类型的chan
   go startServer(addr)

   // in fact, following code is like a simple geerpc client
   cli, _ := client.Dial("tcp", <-addr)//连接服务器
   defer func() { _ = cli.Close() }()

   time.Sleep(time.Second)
   // send options
   var wg sync.WaitGroup
   // send request & receive response
   for i := 0; i < 5; i++ {
      wg.Add(1)
      go func(i int){
         defer wg.Done()
         args:=fmt.Sprintf("geerpc req %d", i)
         var reply string
         if err := cli.Call("Foo.Sum", args, &reply);err != nil{
            log.Fatal("call Foo.Sum error:", err)
         }
         log.Println("reply:", reply)
      }(i)
      wg.Wait()
   }
}
```

------

测试结果大致如下：

>2022/01/24 14:09:23 start rpc server on [::]:50354
>
>2022/01/24 14:09:24 &{Foo.Sum 1 } geerpc req 0
>
>2022/01/24 14:09:24 reply: geerpc resp 1
>
>2022/01/24 14:09:24 &{Foo.Sum 2 } geerpc req 1
>
>2022/01/24 14:09:24 reply: geerpc resp 2
>
>2022/01/24 14:09:24 &{Foo.Sum 3 } geerpc req 2
>
>2022/01/24 14:09:24 reply: geerpc resp 3
>
>2022/01/24 14:09:24 &{Foo.Sum 4 } geerpc req 3
>
>2022/01/24 14:09:24 reply: geerpc resp 4
>
>2022/01/24 14:09:24 &{Foo.Sum 5 } geerpc req 4
>
>2022/01/24 14:09:24 reply: geerpc resp 5

但是默认编码使用gob编码会出现：

>2022/01/24 14:10:24 start rpc server on [::]:50437
>
>2022/01/24 14:10:25 rpc server: read argv err: gob: decoding into local type *string, received remote type interface
>2022/01/24 14:10:25 &{Foo.Sum 1 } 
>
>2022/01/24 14:10:25 reply: geerpc resp 1
>
>2022/01/24 14:10:25 rpc server: read argv err: gob: decoding into local type *string, received remote type interface
>2022/01/24 14:10:25 &{Foo.Sum 2 } 
>
>2022/01/24 14:10:25 reply: geerpc resp 2
>
>2022/01/24 14:10:25 rpc server: read argv err: gob: decoding into local type *string, received remote type interface
>2022/01/24 14:10:25 &{Foo.Sum 3 } 
>
>2022/01/24 14:10:25 reply: geerpc resp 3
>
>2022/01/24 14:10:25 rpc server: read argv err: gob: decoding into local type *string, received remote type interface
>2022/01/24 14:10:25 &{Foo.Sum 4 } 
>
>2022/01/24 14:10:25 reply: geerpc resp 4
>
>2022/01/24 14:10:25 rpc server: read argv err: gob: decoding into local type *string, received remote type interface
>2022/01/24 14:10:25 &{Foo.Sum 5 } 
>
>2022/01/24 14:10:25 reply: geerpc resp 5

不影响通信但还没找到原因是什么

