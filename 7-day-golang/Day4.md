# Day4

前面三天已经完成了编码器，客户端和服务器的基本编码；今天要在原先的基础上，增加超时处理的部分

------

客户端需要处理超时的地方：

* 与服务器建立连接时超时
* 发送请求时写的超时
* 接受回复时读的超时
* 等待服务器回复的超时

服务器要处理的超时：

* 读取客户端请求时读的超时
* 发送回复时写的超时
* 处理请求调用服务的超时

------

`客户端创建链接时的超时：`

将超时限制写入Option结构体里：

service/server.go

```go
type Option struct {
   MagicNumber int        // MagicNumber marks this's a geerpc request
   CodecType   codec.Type // client may choose different Codec to encode body
   ConnectTimeout time.Duration //连接超时 0表示没有限时
   HandleTimeout time.Duration // 处理超时
}
var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	//CodecType:   codec.GobType,
	CodecType:   codec.JsonType,
	ConnectTimeout: time.Second*10,
}
```

有了这层连接限时，只需要给客户端包一层外壳处理超时问题就行，首先解决连接超时：

client使用Dial函数连接服务器，创建一个client实例，所以修改Dial函数来解决超时问题

```go
//发送主要是实现一个Dial函数，调用远端函数
type clientResult struct{
	client *Client
	err error
}
type newClientFunc func(conn net.Conn, opt *service.Option) (client *Client, err error)
func dialTimeout  (f newClientFunc,network,address string,opts ...*service.Option) (client *Client,err error){
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout(network, address,opt.ConnectTimeout) //连接服务器--
	if err != nil {
		return nil, err
	}
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	ch := make(chan clientResult)
	go func(){
		//创建客户端
		client,e := f(conn,opt)
		ch <- clientResult{client,e}
	}()
	if opt.ConnectTimeout == 0{
		result := <- ch
		return result.client, result.err
	}
	select{
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}
func Dial( network,address string,opts ...*service.Option) (client *Client,err error){
	return dialTimeout(NewClient,network,address,opts...)

}
```

改用net.DialTimeout函数，连接超时会返回超时

利用go协程创建client实例，select处理超时，如果超过时间还没有建立成功则返回错误，关闭连接

------

`客户端调用服务的超时:`把后三种超时一起处理了

利用context包来处理调用服务超时：

每次调用设置上下文的超时时间,时间到了会自动调用context.done()给协程发消息

```go
ctx,_ := context.WithTimeout(context.Background(), time.Second)
err := client.Call(ctx, "Foo.Sum", &Args{1, 2}, &reply)
```

用户在调用call时，执行完Go就在等待服务器的执行结果，如果超过执行时间，则自动结束这个调用

```go
func (cli *Client) Call(ctx context.Context,serviceMethod string, args, reply interface{}) error {
   call := cli.Go(serviceMethod, args, reply, make(chan *Call, 1))
   select {
      case <-ctx.Done():
         cli.removeCall(call.Seq)
         return errors.New("rpc client: call failed: " + ctx.Err().Error())
      case call := <-call.Done:
         return call.Error
   }
   return call.Error
}
```

------

`服务器超时`：

也是利用select+chan解决,将option中的handletimeout传入处理请求函数中

开一个go协程处理调用，

主写成用time.after()处理超时

为了防止两次sendResponse；所以利用两个chan，如果在时间内完成call调用，则select进入第二个case，等待发送回复；如果`time.After()` 先于 called 接收到消息，说明处理已经超时，called 和 sent 都将被阻塞。在 `case <-time.After(timeout)` 处调用 `sendResponse`。

```go
s.ServerCodec(f(conn),option.HandleTimeout)
go s.handleRequest(c, req, sending, wg,timeout)//处理请求
---------------------------------------------------------
func (s *Server) handleRequest(c codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration){
   defer wg.Done()
   called := make(chan struct{})
   sent := make(chan struct{})
   go func(){
      err := req.svc.call(req.mtype,req.argv,req.replyv)
      called <- struct{}{} //调用结束
      if err != nil {
         req.h.Error = err.Error()
         s.sendResponse(c, req.h, invalidRequest, sending)
         sent <- struct{}{}//回复结束
         return
      }
      s.sendResponse(c,req.h,req.replyv.Interface(),sending)
      sent <- struct{}{}//回复结束
   }()
   //如果没有时间限制，等待执行完就返回
   if(timeout == 0){
      <- called
      <- sent
      return
   }
   select{
      case <-time.After(timeout):
         req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
         s.sendResponse(c, req.h, invalidRequest, sending)
      case <-called:
         <-sent
   }

}
```

但这里有个问题，建立的sent和called通道是无缓冲的，所以当time.after执行完退出后，协程中`called <- struct{}{} //调用结束` 后阻塞（因为没有人读取了，无缓冲chan需要读和写同时准备好），导致go协程泄漏

