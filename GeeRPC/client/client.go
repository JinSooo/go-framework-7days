package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"geerpc/server"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

/* ------------------------------ RPC Function ------------------------------ */
/**
 * 远程调用函数
 * 		func (t *T) MethodName(argType T1, replyType *T2) error
 */

// 一次RPC调用所需要的信息
type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call // 标志RPC调用请求结束，准备接受响应
}

func (call *Call) done() {
	call.Done <- call
}

/* --------------------------------- Client --------------------------------- */

type Client struct {
	cc       codec.Codec
	option   *server.Option
	header   codec.Header
	seq      uint64           // request id
	pending  map[uint64]*Call // store unfinished requests(requestId -> Call)
	sending  sync.Mutex       // send request mutex
	mutex    sync.Mutex       // client operation mutex
	closing  bool             // client closes the connection
	shutdown bool             // server closes the connection
}

var _ io.Closer = (*Client)(nil)

var errShutdown = errors.New("connection is shut down")

func (client *Client) Close() error {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if client.closing {
		return errShutdown
	}
	client.closing = true
	return client.cc.Close()
}

// client is or not available
func (client *Client) IsAvailable() bool {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	return !client.closing && !client.shutdown
}

// about call fn

func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if client.closing || client.shutdown {
		return 0, errShutdown
	}

	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

func (client *Client) removeCall(seq uint64) *Call {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// terminate all client requests when an error occurs on the server or client
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mutex.Lock()
	defer client.mutex.Unlock()

	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

/**
 * receive response exist three cases
 *  - call 不存在，可能是请求没有发送完整，或者因为其他原因被取消，但是服务端仍旧处理了。
 *  - call 存在，但服务端处理出错，即 h.Error 不为空。
 *  - call 存在，服务端处理正常，那么需要从 body 中读取 Reply 的值。
 */
func (client *Client) receive() {
	var err error

	// wait a response coming in
	for err == nil {
		var header codec.Header
		if err := client.cc.ReadHeader(&header); err != nil {
			break
		}
		call := client.pending[header.Seq]

		// three cases
		switch {
		case call == nil:
			err = client.cc.ReadBody(nil)
		case header.Error != "":
			call.Error = fmt.Errorf(header.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}

	// if there exists any error, the request will be terminated
	client.terminateCalls(err)
}

func NewClient(conn net.Conn, opt *server.Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}

	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error: ", err)
		_ = conn.Close()
		return nil, err
	}

	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *server.Option) *Client {
	client := &Client{
		seq:     1, // seq starts with 1, 0 means invalid call
		cc:      cc,
		option:  opt,
		pending: make(map[uint64]*Call),
	}

	go client.receive()
	return client
}

func parseOptions(opts ...*server.Option) (*server.Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return server.DefaultOption, nil
	}

	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}

	opt := opts[0]
	opt.MagicNumber = server.DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = server.DefaultOption.CodecType
	}
	return opt, nil
}

type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *server.Option) (*Client, error)

// dial with timeout
func dialTimeout(f newClientFunc, network string, address string, opts ...*server.Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout(network, address, opt.ConnectTimeout)
	if err != nil {
		return nil, err
	}

	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()

	// create a goroutine and send result by the channel
	ch := make(chan clientResult)
	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{client: client, err: err}
	}()

	// mean no timeout limit
	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}

	// if it's not done within opt.ConnectTimeout, it will return a error, otherwise, we will get the result
	select {
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}

// Dial connects to an RPC server at the specified network address
func Dial(network string, address string, opts ...*server.Option) (client *Client, err error) {
	return dialTimeout(NewClient, network, address, opts...)
}

// send request

func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)
		// call may be nil, it usually means that Write partially failed,
		// or client has received the response and handled
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Go 是一个异步接口，返回 call 实例。
// Go invokes the function asynchronously.
// It returns the Call structure representing the invocation.
func (client *Client) Go(serviceMethod string, args interface{}, reply interface{}, done chan *Call) *Call {
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
	client.send(call)

	return call
}

// Call 是对 Go 的封装，阻塞 call.Done，等待响应返回，是一个同步接口。
// Call invokes the named function, waits for it to complete,
// and returns its error status.
func (client *Client) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	// call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))

	select {
	/**
	 * client cancel the request
	 * 		ctx, _ := context.WithTimeout(context.Background(), time.Second)
	 * 		var reply int
	 * 		err := client.Call(ctx, "Foo.Sum", &Args{1, 2}, &reply)
	 */
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	// request finish
	case ca := <-call.Done:
		return ca.Error
	}
}

/* ------------------------------ support HTTP ------------------------------ */
/**
 *
 */

// NewHTTPClient new a Client instance via HTTP as transport protocol
func NewHTTPClient(conn net.Conn, opt *server.Option) (*Client, error) {
	// HTTP format
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", server.DefaultRPCPath))

	// Require successful HTTP response, before switching to RPC protocol.
	res, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && res.Status == server.Connected {
		// create rpc client
		return NewClient(conn, opt)
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + res.Status)
	}

	return nil, err
}

// DialHTTP connects to an HTTP RPC server at the specified network address
func DialHTTP(network string, address string, opts ...*server.Option) (client *Client, err error) {
	return dialTimeout(NewHTTPClient, network, address, opts...)
}

// Simplified Dial
// SDial calls different functions to connect to a RPC server
// rpcAddr is a general format (protocol@addr) to represent a rpc server
// eg, http@10.0.0.1:7001, tcp@10.0.0.1:9999, unix@/tmp/geerpc.sock
func SDial(rpcAddr string, opts ...*server.Option) (*Client, error) {
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
