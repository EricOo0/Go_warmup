# Day6

增加简单负载均衡功能

------

在实际的应用中，为了面对高并发场景，同一项服务会由多个服务器提供，客户端根据具体情况选择不同的服务器来提供服务。选择服务器的过程即为负载均衡。

常用的负载均衡算法有：

* 随机选择策略 - 从服务列表中随机选择一个。
* 轮询算法(Round Robin) - 依次调度不同的服务器，每次调度执行 i = (i + 1) mode n。
* 加权轮询(Weight Round Robin) - 在轮询算法的基础上，为每个服务实例设置一个权重，高性能的机器赋予更高的权重，也可以根据服务实例的当前的负载情况做动态的调整，例如考虑最近5分钟部署服务器的 CPU、内存消耗情况。
* 哈希/一致性哈希策略 - 依据请求的某些特征，计算一个 hash 值，根据 hash 值将请求发送到对应的机器。一致性 hash 还可以解决服务实例动态添加情况下，调度抖动的问题。

------

实现负载均衡先要实现服务发现的功能，要使客户端能够获得提供服务的服务器信息

首先定义两个数据结构

* 本次只实现简单的随机选择和轮询round robin算法
* 定义一个Discovery接口，包含服务发现的基本方法

xclient/discovery.go

```go
type SelectMode int

const (
   RandomSelect     SelectMode = iota // select randomly
   RoundRobinSelect                   // select using Robbin algorithm
)

type Discovery interface {
   Refresh() error // refresh from remote registry
   Update(servers []string) error
   Get(mode SelectMode) (string, error)
   GetAll() ([]string, error)
}
```

然后实现这个接口，实现这四个方法

```go
type MultiServersDiscovery struct {
   r       *rand.Rand   // generate random number
   mu      sync.RWMutex // protect following
   servers []string
   index   int // record the selected position for robin algorithm
}

// NewMultiServerDiscovery creates a MultiServersDiscovery instance
func NewMultiServerDiscovery(servers []string) *MultiServersDiscovery {
   d := &MultiServersDiscovery{
      servers: servers,
      r:       rand.New(rand.NewSource(time.Now().UnixNano())),
   }
   d.index = d.r.Intn(math.MaxInt32 - 1)
   return d
}
var _ Discovery = (*MultiServersDiscovery)(nil) //确保 实现了接口

// Refresh doesn't make sense for MultiServersDiscovery, so ignore it
func (d *MultiServersDiscovery) Refresh() error { //由于本次实现的不需要注册中心，所以直接返回
   return nil
}

// Update the servers of discovery dynamically if needed
func (d *MultiServersDiscovery) Update(servers []string) error { //更新服务
   d.mu.Lock()
   defer d.mu.Unlock()
   d.servers = servers
   return nil
}

// Get a server according to mode
func (d *MultiServersDiscovery) Get(mode SelectMode) (string, error) {
   d.mu.Lock()
   defer d.mu.Unlock()
   n := len(d.servers)
   if n == 0 {
      return "", errors.New("rpc discovery: no available servers")
   }
   switch mode {
   case RandomSelect:
      return d.servers[d.r.Intn(n)], nil
   case RoundRobinSelect:
      s := d.servers[d.index%n] // servers could be updated, so mode n to ensure safety
      d.index = (d.index + 1) % n
      return s, nil
   default:
      return "", errors.New("rpc discovery: not supported select mode")
   }
}

// returns all servers in discovery
func (d *MultiServersDiscovery) GetAll() ([]string, error) {
   d.mu.RLock()
   defer d.mu.RUnlock()
   // return a copy of d.servers
   servers := make([]string, len(d.servers), len(d.servers))
   copy(servers, d.servers)
   return servers, nil
}
```

------

服务发现部分完成，接下来向用户提供一个支持负载均衡的客户端

```go
type XClient struct {
   d       Discovery
   mode    SelectMode
   opt     *service.Option
   mu      sync.Mutex // protect following
   clients map[string]*client.Client
}

var _ io.Closer = (*XClient)(nil)

func NewXClient(d Discovery, mode SelectMode, opt *service.Option) *XClient {
   return &XClient{d: d, mode: mode, opt: opt, clients: make(map[string]*client.Client)}
}

func (xc *XClient) Close() error {
   xc.mu.Lock()
   defer xc.mu.Unlock()
   for key, client := range xc.clients {
      // I have no idea how to deal with error, just ignore it.
      _ = client.Close()
      delete(xc.clients, key)
   }
   return nil
}

func (xc *XClient) dial(rpcAddr string) (*client.Client, error) {
   xc.mu.Lock()
   defer xc.mu.Unlock()
   cli, ok := xc.clients[rpcAddr] //rpc服务地址
   if ok && !cli.IsAvailable() {
      _ = cli.Close()
      delete(xc.clients, rpcAddr)
      cli = nil
   }
   if cli == nil {
      var err error
      cli, err = client.XDial(rpcAddr, xc.opt)
      if err != nil {
         return nil, err
      }
      xc.clients[rpcAddr] = cli
   }
   return cli, nil
}

func (xc *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
   client, err := xc.dial(rpcAddr)
   if err != nil {
      return err
   }
   return client.Call(ctx, serviceMethod, args, reply)
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
// xc will choose a proper server.
//负载均衡的CALL
func (xc *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
   rpcAddr, err := xc.d.Get(xc.mode)
   if err != nil {
      return err
   }
   return xc.call(rpcAddr, ctx, serviceMethod, args, reply)
}
```

调用客户端进行Call会先获取一个服务器的地址，然后调用正常client的Call

给xclient增加一个广播功能，调用所有提供服务的server，只返回其中一个结果

```go
// Broadcast invokes the named function for every server registered in discovery
func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
   servers, err := xc.d.GetAll()
   if err != nil {
      return err
   }
   var wg sync.WaitGroup
   var mu sync.Mutex // protect e and replyDone
   var e error
   replyDone := reply == nil // if reply is nil, don't need to set value
   ctx, cancel := context.WithCancel(ctx)
   for _, rpcAddr := range servers { //遍历所有server
      wg.Add(1)
      go func(rpcAddr string) {
         defer wg.Done()
         var clonedReply interface{}
         if reply != nil {
            clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface() //利用反射获得reply的类型
         }
         err := xc.call(rpcAddr, ctx, serviceMethod, args, clonedReply) //进行调用
         mu.Lock()
         if err != nil && e == nil {
            e = err
            cancel() // if any call failed, cancel unfinished calls
         }
         if err == nil && !replyDone {
            reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
            replyDone = true
         }
         mu.Unlock()
      }(rpcAddr)
   }
   wg.Wait()
   return e
}
```

------

Demo

```go
package main

import (
   "context"
   "geerpc/service"
   "geerpc/xclient"
   "log"
   "net"
   "sync"
   "time"
)
//服务注册
type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
   *reply = args.Num1 + args.Num2
   return nil
}

func (f Foo) Sleep(args Args, reply *int) error {
   time.Sleep(time.Second * time.Duration(args.Num1))
   *reply = args.Num1 + args.Num2
   return nil
}
//启动服务器
func startServer(addrCh chan string) {
   var foo Foo
   l, _ := net.Listen("tcp", ":0")
   server := service.NewServer()
   _ = server.Register(&foo)
   addrCh <- l.Addr().String()
   server.Accept(l)
}
func foo(xc *xclient.XClient, ctx context.Context, typ, serviceMethod string, args *Args) {
   var reply int
   var err error
   switch typ {
   case "call":
      err = xc.Call(ctx, serviceMethod, args, &reply)
   case "broadcast":
      err = xc.Broadcast(ctx, serviceMethod, args, &reply)
   }
   if err != nil {
      log.Printf("%s %s error: %v", typ, serviceMethod, err)
   } else {
      log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
   }
}
func call(addr1, addr2 string) {
   d := xclient.NewMultiServerDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
   xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
   defer func() { _ = xc.Close() }()
   // send request & receive response
   var wg sync.WaitGroup
   for i := 0; i < 5; i++ {
      wg.Add(1)
      go func(i int) {
         defer wg.Done()
         foo(xc, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
      }(i)
   }
   wg.Wait()
}

func broadcast(addr1, addr2 string) {
   d := xclient.NewMultiServerDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
   xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
   defer func() { _ = xc.Close() }()
   var wg sync.WaitGroup
   for i := 0; i < 5; i++ {
      wg.Add(1)
      go func(i int) {
         defer wg.Done()
         foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
         // expect 2 - 5 timeout
         ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
         foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
      }(i)
   }
   wg.Wait()
}


func main() {
   log.SetFlags(0)
   ch1 := make(chan string)
   ch2 := make(chan string)
   // start two servers 启动两个服务器
   go startServer(ch1)
   go startServer(ch2)

   addr1 := <-ch1
   addr2 := <-ch2

   time.Sleep(time.Second)
   call(addr1, addr2) //调用一项服务
   broadcast(addr1, addr2) //调用所有服务
}
```

call 是利用负载均衡调用一个服务器的服务

broadcast调用所有服务器执行一遍