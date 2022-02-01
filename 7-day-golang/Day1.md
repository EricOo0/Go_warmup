# Day1

今天主要是实现消息的序列化和反序列化；以及实现一个简易的服务端，仅接受消息

------

一个RPC调用主要需要服务名，方法名，请求参数和回复参数

```go
err = client.Call("Service.Method", &args, &reply)
```

所以首先需要定义一个Header结构体，其包包含了： 

* 调用的函数和方法名
* 用于区分不同请求的序列号
* 错误信息err

参数args和reply由于不同的rpc调用有不同的结构，所以在后续的body部分传输。

服务器和客户端要通信需要确定通信的编码格式，所以先抽象出来一个编码器coderc的接口;

Codec接口需要实现下列函数：

* ReadHeader函数读取消息头部
* ReadBody函数读取消息内容
* Write 写回复
* Close 关闭连接

Day1/codec/codec.go

```go
package codec
import "io"

type Header struct{
	ServiceMethod string  //"调用方法 格式 service.method"
	Seq uint64   //客户端选择的序列号
	Error string //服务端如果出错了就吧错误信息放到error中
}

//编码器是一个接口，需要实现:关闭数据流，读，写等方法
type Codec interface{
	io.Closer //关闭数据流的接口，即需要一个close函数
	ReadBody(interface{}) error
	ReadHeader(*Header) error
	Write(*Header,interface{}) error // 写-写头部和body
}

//codec 的构造函数,传入一个io.readwritecloser 实例，返回一个codec实例
type NewCodecFunc func(io.ReadWriteCloser) Codec
type Type string

//不同的序列化方法
const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // not implemented
)
//根据不同的序列化类型选择不同的构造函数
var NewCodecFuncMap map[Type]NewCodecFunc
func init(){
	NewCodecFuncMap = make(map[Type]NewCodecFunc) //string-func 的map
	NewCodecFuncMap[GobType] = NewGobCodec //新建一个Gob编码器
}
```

init 是golang特殊的函数，先于main函数执行，会自动调用，一个包可以有多个init函数，不能被调用。

------

不同的序列化方法都需要实现codec这个接口

Gob是Go语言自己的以二进制形式序列化和反序列化程序数据的格式；可以在encoding 包中找到。

Day1/codec/gob.go

```go
package codec

import (
   "bufio"
   "encoding/gob"
   "io"
   "log"
)

//gob实现了codec接口
type GobCodec struct {
   conn  io.ReadWriteCloser
   enc *gob.Encoder
   dec *gob.Decoder
   buf *bufio.Writer
}

//GobCodec 的构造方法
func NewGobCodec(conn io.ReadWriteCloser) Codec{
   buff := bufio.NewWriter(conn)
   return &GobCodec{
      conn:conn,
      buf:buff,
      enc:gob.NewEncoder(buff), //编码到buff里
      dec:gob.NewDecoder(conn),  //从conn解码
   }
}

//实现GobCodec的读写方法

func (c * GobCodec) ReadHeader(h *Header) error {
   return c.dec.Decode(h)
}
func (c *GobCodec) ReadBody(body interface{}) error{
   return c.dec.Decode(body)
}
func (c * GobCodec) Write(h *Header,body interface{}) (err error){
   // 写完要从buf里flush到io然后关闭
   defer func(){
      _ =c.buf.Flush()
      if err !=nil{
         _ = c.Close()
      }
   }()

   if err:= c.enc.Encode(h);err !=nil{
      log.Println("rpc codec: gob error encoding header:", err)
      return err
   }
   if err := c.enc.Encode(body); err != nil {

      log.Println("rpc codec: gob error encoding body:", err)
      return err
   }
   return  nil
}
func (c *GobCodec) Close() error{
   return c.conn.Close()

}
```

Day1/codec/json.go

json的实现和gob一样

```go
package codec

import (
   "bufio"
   "encoding/json"
   "io"
   "log"
)

type JsonCodec struct {
   conn  io.ReadWriteCloser
   enc *json.Encoder
   dec * json.Decoder
   buf *bufio.Writer
}
func NewJsonCodec(conn io.ReadWriteCloser) Codec{
   buff := bufio.NewWriter(conn)
   return &JsonCodec{
      conn:conn,
      buf:buff,
      enc:json.NewEncoder(buff), //编码到buff里
      dec:json.NewDecoder(conn), //从conn解码
   }
}

//实现JsonCodec的读写方法
func (c * JsonCodec) ReadHeader(h *Header) error {
   err :=c.dec.Decode(h)
    return err
}
func (c *JsonCodec) ReadBody(body interface{}) error{
   return c.dec.Decode(body)
}
func (c * JsonCodec) Write(h *Header,body interface{}) (err error){
   // 写完要从buf里flush到io然后关闭
   defer func(){

      _ =c.buf.Flush()
      if err !=nil{
         _ = c.Close()
      }

   }()
   if err:= c.enc.Encode(h);err !=nil{
      log.Println("rpc codec: json error encoding header:", err)
      return err
   }
   if err := c.enc.Encode(body); err != nil {
      log.Println("rpc codec: json error encoding body:", err)
      return err
   }
   return  nil
}
func (c *JsonCodec) Close() error{
   return c.conn.Close()

}
```

------

协议格式：

客户端和服务器通信内容主要按照以下方式进行序列化

* Option部分采用固定编码格式，Option决定了后面消息内容使用什么编码格式
* Header包含了调用的方法和服务名以及序列号
* body包含了参数argv和回复replyv的结构体

> | Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
> | <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|

决定了通信协议并且实现了编码器接口后，可以写一个简单的server来负责接收客户端的请求并进行解码和回复

Day1/server.go

```go
package geerpc

import (
   "encoding/json"
   "fmt"
   "geerpc/codec"
   "io"
   "log"
   "net"
   "reflect"
   "sync"
)

const MagicNumber = 0x3bef5c

//option 用于决定通信协议类型
//整个RPC编码的格式如下
//| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
//| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
type Option struct {
   MagicNumber int        // MagicNumber marks this's a geerpc request
   CodecType   codec.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
   MagicNumber: MagicNumber,
   CodecType:   codec.GobType,
   //CodecType:   codec.JsonType,
}

type Server struct {}

func NewServer() *Server{
   return &Server{}
}
// DefaultServer is the default instance of *Server. 默认的server，方便一会测试
var DefaultServer = NewServer()

// 服务端的任务就是接受请求，处理和回复请求
func (s *Server) Accept(lis net.Listener){
   // 无限循环一直监听网络
   for{
      conn,err := lis.Accept()
      if err!=nil{
         log.Println("rpc server: accept error:", err)
         return
      }
      //每接受一个连接开一个goroutine处理请求
      go s.ServerConn(conn)
   }
}
// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

// 消息的格式可能是如下| Option | Header1 | Body1 | Header2 | Body2 | ...
func (s *Server) ServerConn(conn io.ReadWriteCloser) {
   //先decode Option
   var option Option
  //if err := gob.NewDecoder(conn).Decode(&option); err!=nil
   if err := json.NewDecoder(conn).Decode(&option); err!=nil{
      log.Println("rpc server: options error: ", err)
      return
   }
   if option.MagicNumber != MagicNumber {
      log.Printf("rpc server: invalid magic number %x", option.MagicNumber)
      return
   }
   f := codec.NewCodecFuncMap[option.CodecType] //根据编码类型选择编码器初始化函数
   if f == nil{
      log.Printf("rpc server: invalid codec type %s", option.CodecType)
      return
   }
   s.ServerCodec(f(conn))
}
// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{ }{}
func (s *Server) ServerCodec( c codec.Codec){
   sending := new(sync.Mutex) // 添加互斥锁保证每条回复完整发送
   wg := new(sync.WaitGroup)  // wait until all request are handled

   for{
      req,err := s.readRequest(c)//读请求
      if err !=nil{
         if req == nil{
            break
         }
         req.h.Error = err.Error()
         s.sendResponse(c, req.h, invalidRequest, sending)
         continue
      }
      wg.Add(1)
      go s.handleRequest(c, req, sending, wg)//处理请求
   }
   //没有请求了会跳出循环
   wg.Wait()
   _ = c.Close()
}

//serveCodec 的过程
//读取请求 readRequest
//处理请求 handleRequest
//回复请求 sendResponse
type  request struct{
   h *codec.Header
   argv,replyv reflect.Value
}
func (s *Server) readRequstHeader(c codec.Codec) (*codec.Header,error){
   var h codec.Header
   if err:= c.ReadHeader(&h) ; err!=nil{
      fmt.Println(err)
      if err != io.EOF && err != io.ErrUnexpectedEOF {

         log.Println("rpc server: read header error:", err)
      }
      return nil, err
   }
   return &h,nil
}
func (s *Server) readRequest(c codec.Codec) (*request, error){
   //读头部
   h,err:=s.readRequstHeader(c)
   if err != nil {
      return nil, err
   }
   req := &request{h: h}
   // TODO: now we don't know the type of request argv
   // day 1, just suppose it's string暂时现使用string作为参数
   req.argv = reflect.New(reflect.TypeOf(""))
   //读body
   if err = c.ReadBody(req.argv.Interface()); err != nil {
      log.Println("rpc server: read argv err:", err)
   }
   return req, nil
}
func (s *Server) handleRequest(c codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup){
   defer wg.Done()
   log.Println(req.h, req.argv.Elem())
   req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
   s.sendResponse(c,req.h,req.replyv.Interface(),sending)
}
func (s* Server) sendResponse(c codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex){
   sending.Lock()
   defer sending.Unlock()
   if err := c.Write(h, body); err != nil {
      log.Println("rpc server: write response error:", err)
   }
}
```

>sync.WaitGroup 的作用：用于阻塞等待一组Go 程的结束
>
>​	主 Go 程调用 Add() 来设置等待的 Go 程数，然后该组中的每个 Go 程都需要在运行结束时调用 Done()， 递减 WaitGroup 的 Go 程计数器 counter。当 counter 变为 0 时，主 Go 程（需要在调用wait()）被唤醒继续执行。	
>
>​	waitgroup可以让住go程等待所有子go程接受然后继续执行
>
>request结构体包含了请求头和传递的参数，参数是动态变化的所以采用reflect包来定义
>
>​	type  request struct{
>
>​	  	 h *codec.Header
>   		argv,replyv reflect.Value
>​	}
>
>releact反射机制：
>
>​	对于每个变量都有{Type,Value}两个部分，通过TypeOf和ValueOf可以获取

------

简单的接受rpc的server已经写好了，server会一直监听并且并发地处理请求，接下来写一个main函数测试一下

Day1/main/main.go

```go
package main

import (
   "encoding/gob"
   "encoding/json"
   "fmt"
   "geerpc"
   "geerpc/codec"
   "log"
   "net"
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
   geerpc.Accept(l) //调用服务器监听客户端
}

func main() {
   addr := make(chan string) //addr是一个 string类型的chan
   go startServer(addr) //一个go协程启动一个server， addr是个通道将服务器信息传出来

  //下面是模拟的客户端
   // in fact, following code is like a simple geerpc client
   conn, _ := net.Dial("tcp", <-addr)//连接服务器
   defer func() { _ = conn.Close() }()

   time.Sleep(time.Second)
   // send options
// 消息的格式可能是如下| Option | Header1 | Body1 | Header2 | Body2 | ...
   _ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)  //先发送option
   //_ = gob.NewEncoder(conn).Encode(geerpc.DefaultOption)  
   time.Sleep(time.Second)  //important！  如果option和header&body用一种编码方式，可能合成一个包发送过去了，server那边解码option的时候把header忽略了，导致server在等header，client在等reply阻塞住
   //cc := codec.NewJsonCodec(conn)
   cc := codec.NewGobCodec(conn)
   // send request & receive response
   for i := 0; i < 5; i++ {
      h := &codec.Header{
         ServiceMethod: "Foo.Sum",
         Seq:           uint64(i),
         Error: "",
      }
      t:=fmt.Sprintf("geerpc req %d", h.Seq)
      _ = cc.Write(h, t)
      _ = cc.ReadHeader(h)
      var reply string
      _ = cc.ReadBody(&reply)
      log.Println("reply:", reply)
   }
}
```