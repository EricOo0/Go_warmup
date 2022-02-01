package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"geerpc/service"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

//定义一个结构体来存储一次rpc调用需要的所有信息

type Call struct{
	Seq uint64 //序列号，标志一次请求
	ServiceMethod string //请求服务和方法
	Args interface{}
	Reply interface{}
	Error error
	Done chan *Call //当一次调用完成，用于通知调用方

}
//支持异步调用，使用channel来通知调用方
func (call *Call) done() {
	call.Done <- call
}


//对于一个client,可能会有多个调用
type Client struct{
	c codec.Codec
	h codec.Header
	seq uint64
	opt *service.Option
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

// 一个client需要实现三个函数
//注册调用
//删除调用
//结束调用

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
//对一个客户端端来说，接收响应、发送请求是最重要的 2 个功能。
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

//创建Client实例
func NewClient(conn net.Conn,opt *service.Option) (*Client,error){
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
func newClientCodec(c codec.Codec,opt *service.Option) *Client{
	client := &Client{
		c:c,
		seq:1,
		opt:opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

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
func parseOptions(opts ...*service.Option)(*service.Option,error){


	if len(opts) == 0 || opts[0] == nil {
		return service.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = service.DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = service.DefaultOption.CodecType
	}

	return opt, nil
}

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


//下面的Go和Call是客户端暴露出来的Rpc调用接口

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

func (cli *Client) Call(ctx context.Context,serviceMethod string, args, reply interface{}) error {
	call := cli.Go(serviceMethod, args, reply, make(chan *Call, 1))
	select {
		case <-ctx.Done():
			cli.removeCall(call.Seq)
			return errors.New("rpc client: call failed: " + ctx.Err().Error())
		case call := <-call.Done:
			return call.Error
	}
}

//HTTP
// NewHTTPClient new a Client instance via HTTP as transport protocol
func NewHTTPClient(conn net.Conn, opt *service.Option) (*Client, error) {
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", service.DefaultRPCPath))

	// Require successful HTTP response
	// before switching to RPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == service.Connected {
		return NewClient(conn, opt)
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

func XDial(rpcAddr string, opts ...*service.Option) (*Client, error) {
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