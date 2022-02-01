package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

const MagicNumber = 0x3bef5c
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

//查找服务名
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
// 一个请求需要的所有信息
type  request struct{
	h *codec.Header
	argv,replyv reflect.Value
	mtype *methodType
	svc *service
}

//option 用于决定通信协议类型
//整个RPC编码的格式如下
//| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
//| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
type Option struct {
	MagicNumber int        // MagicNumber marks this's a geerpc request
	CodecType   codec.Type // client may choose different Codec to encode body
	ConnectTimeout time.Duration //连接超时
	HandleTimeout time.Duration // 处理超时
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	//CodecType:   codec.GobType,
	CodecType:   codec.JsonType,
	ConnectTimeout: time.Second*10,
}

func NewServer() *Server {
	return &Server{}
}

// 服务端的任务就是接受请求，处理和回复请求
func (s *Server) Accept(lis net.Listener){
	// 无限循环一直监听网络
	for{
		conn,err := lis.Accept()
		if err!=nil{
			log.Println("rpc server: accept error:", err)
			return
		}
		//没接受一个连接开一个goroutine处理请求

		go s.ServerConn(conn)
	}
}
// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

func (s *Server) ServerConn(conn io.ReadWriteCloser) {
	//先decode Option
	var option Option
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
	s.ServerCodec(f(conn),option.HandleTimeout)
}
// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{ }{}
func (s *Server) ServerCodec( c codec.Codec,timeout time.Duration){
	sending := new(sync.Mutex) // 添加互斥锁保证完整发送
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
		go s.handleRequest(c, req, sending, wg,timeout)//处理请求
	}
	//没有请求了会跳出循环
	wg.Wait()
	_ = c.Close()
}

//serveCodec 的过程
//读取请求 readRequest
//处理请求 handleRequest
//回复请求 sendResponse

func (s *Server) readRequstHeader(c codec.Codec) (*codec.Header,error){
	var h codec.Header
	if err:= c.ReadHeader(&h) ; err!=nil{

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
	//根据header确认要请求的服务和方法
	req.svc,req.mtype,err = s.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.argv = req.mtype.newArgv()
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
func (s*Server) sendResponse(c codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex){
	sending.Lock()
	defer sending.Unlock()
	if err := c.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

//HTTP
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