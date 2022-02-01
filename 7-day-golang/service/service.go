package service

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

type service struct{
	name string
	typ reflect.Type
	rcvr reflect.Value
	method map[string] *methodType
}
type methodType struct{
	method reflect.Method
	ArgType reflect.Type
	ReplyType reflect.Type
	numCalls uint64
}
func (m *methodType)  NumCalls() uint64{
	return atomic.LoadUint64(&m.numCalls)
}
func (m *methodType) newArgv() reflect.Value{
	var argv reflect.Value
	if m.ArgType.Kind() == reflect.Ptr{
		argv = reflect.New(m.ArgType.Elem())
	}else{
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}
func (m *methodType) newReplyv() reflect.Value  {
	//reply 应该是指针
	replyv := reflect.New(m.ReplyType.Elem())
	switch m.ReplyType.Elem().Kind(){
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(),0,0))
	}
	return replyv
}

func newService(rcvr interface{}) *service{
	s := new(service)
	//利用反射获得服务的值和名字等信息

	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()

	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods()
	return s
}
func (s *service)registerMethods(){
	//注册服务的所有方法
	s.method = make(map[string]*methodType)
	for i:=0;i<s.typ.NumMethod();i++{

		method := s.typ.Method(i)
		mtype := method.Type
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

func (s *service) call(m *methodType,argv,replyv reflect.Value) error{
	atomic.AddUint64(&m.numCalls,1)
	f := m.method.Func
	returnValue := f.Call([]reflect.Value{s.rcvr,argv,replyv})
	if errInter := returnValue[0].Interface(); errInter!=nil{
		return errInter.(error)
	}
	return nil
}