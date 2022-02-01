# Day3

今天主要是利用golang的反射机制在服务器端实现服务注册的功能

------

Rpc的基本能力是让客户端像调用本地程序一样调用远程程序，所以在完成了编码器codec和客户端client后，需要在服务端先抽象出服务service和methodType两个结构体；

* methodType存储一个方法的参数类型，调用次数，函数入口等
* service存储服务名，拥有的方法map，服务的类型和实例等

service/service.go

```go
type service struct{
   name string //服务名称
   typ reflect.Type // 服务的类型。如*main.Foo
   rcvr reflect.Value // 注册的服务实例 如 var foo Foo，rcvr就是foo
   method map[string] *methodType //一个服务有多个函数可以调用，每个函数用一个method type存储
}
type methodType struct{
   method reflect.Method  //方法
   ArgType reflect.Type   //rpc的参数
   ReplyType reflect.Type //rpc的回复
   numCalls uint64  //调用次数
}
```

服务需要注册到服务器才能被远程调用，通过这两个结构体，我们可以利用反射机制在服务端存储各项服务以及其拥有的方法和参数要求。

首先要把服务注册到服务端：

通过newService函数将服务注册到服务端

例如：我们有一个Foo结构，Foo有一个成员函数sum：Foo{ func sum();}；

那么我们要注册的时候就是新建一个foo实例，然后调用newService将foo注册到服务器，foo就是rcvr实例

```go
func newService(rcvr interface{}) *service{
   s := new(service)
   //利用反射获得服务的值和名字等信息
   s.rcvr = reflect.ValueOf(rcvr) //foo实例
   s.name = reflect.Indirect(s.rcvr).Type().Name() //Foo
   s.typ = reflect.TypeOf(rcvr) //*main.Foo
   if !ast.IsExported(s.name) { //这里是做一个验证，调用的方法应该是导出方法，所以得是大写字母开头
      log.Fatalf("rpc server: %s is not a valid service name", s.name)
   }
   s.registerMethods() //注册服务的所有的方法到这个服务的map中
   return s
}
func (s *service)registerMethods(){
   //注册服务的所有方法
   s.method = make(map[string]*methodType)
   for i:=0;i<s.typ.NumMethod();i++{ //一共有s.typ.NumMethod()个方法
      method := s.typ.Method(i) 
     mtype := method.Type //函数类型，入参应该是三个,第一个是类实例，类似this指针--method(rcvr,argv,replyv) error{}
      if mtype.NumIn()!=3 ||mtype.NumOut()!=1{
         continue;
      }
      if mtype.Out(0) != reflect.TypeOf((*error)(nil)).Elem(){
         continue;
      }
     
      argType := mtype.In(1)
      replyType := mtype.In(2)

      if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
         continue
      }
      s.method[method.Name] = &methodType{
         method:  method,
         ArgType: argType,
         ReplyType: replyType,
      }
      log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
   }
}
func isExportedOrBuiltinType(t reflect.Type) bool {
   return ast.IsExported(t.Name()) || t.PkgPath() == ""
}
```

服务注册好后还要实现一个调用的方法call

```go
func (s *service) call(m *methodType,argv,replyv reflect.Value) error{
   atomic.AddUint64(&m.numCalls,1) //调用次数+1
   f := m.method.Func
   returnValue := f.Call([]reflect.Value{s.rcvr,argv,replyv})
   if errInter := returnValue[0].Interface(); errInter!=nil{
      return errInter.(error)
   }
   return nil
}
```

完成服务的注册后，这个服务和其方法就可以调用了。

客户端向服务器发出请求，在处理请求的时候，根据不同的函数methodType，我们就能构造不同的请求结构体argv和回复结构体replyv用于处理请求：

m为客户端调用的方法类型，通过反射，知道方法就能知道方法的参数信息

```go
func (m *methodType) newArgv() reflect.Value{
   var argv reflect.Value
  // args如果是指针需要特殊处理
   if m.ArgType.Kind() == reflect.Ptr{
      argv = reflect.New(m.ArgType.Elem()) 
   }else{
      argv = reflect.New(m.ArgType).Elem()
   }
   return argv//返回的argv是值，不是指针
}
func (m *methodType) newReplyv() reflect.Value  {
   //传入的reply 应该是指针
   replyv := reflect.New(m.ReplyType.Elem())
   switch m.ReplyType.Elem().Kind(){
     //如果reply是map或slice要初始化一下
   case reflect.Map:
      replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
   case reflect.Slice:
      replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(),0,0))
   }
   return replyv
}
func (m *methodType)  NumCalls() uint64{
  //64位无符号数，要使用原子操作
	return atomic.LoadUint64(&m.numCalls)
}
```

现在通过反射我们已经将其映射为服务了，但处理请求的过程还没完成

从接收到请求到回复还差以下几个步骤：第一步，根据入参类型，将请求的 body 反序列化；第二步，调用 `service.call`，完成方法调用；第三步，将 reply 序列化为字节流，构造响应报文，返回。

修改原来的server.go

```go
type Server struct {
   serviceMap sync.Map
}
//注册服务到server里
func (server *Server)Register(rcvr interface{}) error{
	s := newService(rcvr)
	if _,dup := server.serviceMap.LoadOrStore(s.name,s);dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}
// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer()
//注册一个默认的方便使用
func Register(rcvr interface{}) error { return DefaultServer.Register(rcvr) }

//根据客户端发来的方法名来查找服务名
func (server *Server) findService(serviceMethod string)(svc *service , mtype *methodType,err error){
	//根据 service.method招服务
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}

```

修改readRequest方法，根据找到的服务和方法生成入参类型

```go
// 一个请求需要的所有信息
type  request struct{
   h *codec.Header
   argv,replyv reflect.Value
   mtype *methodType
   svc *service
}
func (s *Server) readRequest(c codec.Codec) (*request, error){
	//读头部
	h,err:=s.readRequstHeader(c)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	//根据header确认要请求的服务和方法的类型和服务
	req.svc,req.mtype,err = s.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.argv = req.mtype.newArgv() // 得到入参的类型
	req.replyv = req.mtype.newReplyv()

	// readbody需要传入一个指针
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	//读body
	if err = c.ReadBody(argvi); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}
//处理请求 即为 调用对应的函数
func (s *Server) handleRequest(c codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup){
	defer wg.Done()

	err := req.svc.call(req.mtype,req.argv,req.replyv) //调用函数
	if err != nil {
		req.h.Error = err.Error()
		s.sendResponse(c, req.h, invalidRequest, sending)
		return
	}
	s.sendResponse(c,req.h,req.replyv.Interface(),sending)
}
```

此时我们的服务注册就完成了

------

下面修改main函数测试一下服务注册

```go
package main

import (
   "geerpc/client"
   "geerpc/service"
   "log"
   "net"
   "sync"
   "time"
)

//要注册的服务和方法
type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
   *reply = args.Num1 + args.Num2
   return nil
}

func startServer(addr chan string) {
   // pick a free port
   var foo Foo
  //注册到服务器去
   if err := service.Register(&foo);err !=nil{
      log.Fatal("register error:", err)
   }
	//启动服务器
   l, err := net.Listen("tcp", ":0")
   if err != nil {
      log.Fatal("network error:", err)
   }
   log.Println("start rpc server on", l.Addr())
   addr <- l.Addr().String()
   service.Accept(l)
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
         args:=&Args{Num1: i,Num2: i*i}
         var reply int
         if err := cli.Call("Foo.Sum", args, &reply); err != nil {
            log.Fatal("call Foo.Sum error:", err)
         }
         log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
      }(i)
      wg.Wait()
   }
}
```

结果：

>2022/01/25 16:14:34 rpc server: register Foo.Sum. 
>
>2022/01/25 16:14:34 start rpc server on [::]:52428
>
>2022/01/25 16:14:35 0 + 0 = 0
>
>2022/01/25 16:14:35 1 + 1 = 2
>
>2022/01/25 16:14:35 2 + 4 = 6
>
>2022/01/25 16:14:35 3 + 9 = 12
>
>2022/01/25 16:14:35 4 + 16 = 20