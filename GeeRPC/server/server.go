package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"go/ast"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/* --------------------------------- 需要协商的内容 -------------------------------- */
/**
 * | Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
 * | <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   -------> |
 *
 * Option 固定在报文的最开始，Header 和 Body 可以有多个
 * | Option | Header1 | Body1 | Header2 | Body2 | ...
 */

const MagicNumber = 0x3bef5c

/**
 * timeout 超时处理：
 * 	需要客户端处理超时的地方有：
 * 		与服务端建立连接，导致的超时
 * 		发送请求到服务端，写报文导致的超时
 * 		等待服务端处理时，等待处理导致的超时（比如服务端已挂死，迟迟不响应）
 *
 * 		从服务端接收响应时，读报文导致的超时
 * 	需要服务端处理超时的地方有：
 * 		读取客户端请求报文时，读报文导致的超时
 * 		发送响应报文时，写报文导致的超时
 * 		调用映射服务的方法时，处理报文导致的超时
 */

type Option struct {
	MagicNumber    int           // MagicNumber marks this is a geerpc request
	CodecType      codec.Type    // client chooses the codec type
	ConnectTimeout time.Duration // 0 means no limit
	HandleTimeout  time.Duration
}

var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodecType:      codec.GobType,
	ConnectTimeout: time.Second * 10,
}

/* --------------------------------- Server --------------------------------- */

type Server struct {
	serviceMap sync.Map
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = &Server{}

// Register publishes in the server the set of methods
func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

func Register(rcvr interface{}) error { return DefaultServer.Register(rcvr) }

func (server *Server) findService(serviceMethod string) (serv *service, mtype *methodType, err error) {
	// find the dot and then split the serviceMethod
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}

	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svc, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}

	serv = svc.(*service)
	mtype = serv.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}

// loop waiting for a socket connection
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		fmt.Println("get a conn")

		go server.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

// ServeConn runs the server on a single connection
// get option instance and process negotiation logic
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { conn.Close() }()

	// get option form client
	var option Option
	if err := json.NewDecoder(conn).Decode(&option); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if option.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", option.MagicNumber)
		return
	}

	fn := codec.NewCodecFuncMap[option.CodecType]
	if fn == nil {
		log.Printf("rpc server: invalid codec type %s", option.CodecType)
		return
	}

	server.serveCodec(fn(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	// make sure to send a complete response
	sending := new(sync.Mutex)
	// wait until all requests are handled
	wg := new(sync.WaitGroup)

	for {
		// read request
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			// send response
			// return error to client
			req.header.Error = err.Error()
			server.sendResponse(cc, req.header, invalidRequest, sending)
			return
		}

		// handle request
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg, time.Second * 10)
	}
}

/* --------------------------------- request -------------------------------- */

type request struct {
	header *codec.Header // header of request
	argv   reflect.Value
	replyv reflect.Value
	serv   *service
	mtype  *methodType
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var header codec.Header

	if err := cc.ReadHeader(&header); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}

	return &header, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	header, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}

	req := &request{header: header}
	req.serv, req.mtype, err = server.findService(header.ServiceMethod)
	if err != nil {
		return req, err
	}

	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// make sure that argvi is a pointer, ReadBody need a pointer as parameter
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	if err := cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read argv err:", err)
		return req, err
	}

	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, header *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()

	if err := cc.Write(header, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()

	called := make(chan struct{})
	sent := make(chan struct{})

	go func() {
		err := req.serv.call(req.mtype, req.argv, req.replyv)
		called <- struct{}{}

		if err != nil {
			req.header.Error = err.Error()
			server.sendResponse(cc, req.header, invalidRequest, sending)
			sent <- struct{}{}
			return
		}
		server.sendResponse(cc, req.header, req.replyv.Interface(), sending)
		sent <- struct{}{}
	}()

	// mean no timeout limit
	if timeout == 0 {
		<-called
		<-sent
		return
	}

	select {
	case <-time.After(timeout):
		req.header.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		server.sendResponse(cc, req.header, invalidRequest, sending)
	case <-called:
		<-sent
	}
}

/* -------------------------------- 结构体映射为服务 -------------------------------- */
/**
 * 一个函数需要能够被远程调用，需要满足如下五个条件：
 *
 * - the method’s type is exported. – 方法所属类型是导出的。
 * - the method is exported. – 方式是导出的。
 * - the method has two arguments, both exported (or builtin) types. – 两个入参，均为导出或内置类型。
 * - the method’s second argument is a pointer. – 第二个入参必须是一个指针。
 * - the method has return type error. – 返回值为 error 类型。
 * 即
 * func (t *T) MethodName(argType T1, replyType *T2) error
 */

// Method

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	numCalls  uint64 // method call times
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value

	// arg may be a pointer type, or a value type
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}

	return argv
}

func (m *methodType) newReplyv() reflect.Value {
	// reply must be a pointer type
	replyv := reflect.New(m.ReplyType.Elem())

	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}

	return replyv
}

// Service

// a struct map a service
type service struct {
	name   string                 // structName
	typ    reflect.Type           // structType
	rcvr   reflect.Value          // original struct
	method map[string]*methodType // all methods that meet the conditions
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()

	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}

	s.registerMethod()
	return s
}

// register all available method
func (s *service) registerMethod() {
	s.method = make(map[string]*methodType)

	// use reflect
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type

		// this is a necessary condition that method must meet
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}

		argType := mType.In(1)
		replyType := mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}

		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
	}
}

func isExportedOrBuiltinType(typ reflect.Type) bool {
	return ast.IsExported(typ.Name()) || typ.PkgPath() == ""
}

// call method
func (s *service) call(m *methodType, argv reflect.Value, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)

	fn := m.method.Func
	returnValues := fn.Call([]reflect.Value{s.rcvr, argv, replyv})

	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
