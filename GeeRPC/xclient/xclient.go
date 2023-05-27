package xclient

import (
	"context"
	"geerpc/client"
	"geerpc/server"
	"io"
	"reflect"
	"sync"
)

/* ------------------------------- 支持负载均衡的客户端 ------------------------------- */

type XClient struct {
	mutex     sync.Mutex
	discovery Discovery
	mode      SelectMode
	opt       *server.Option
	clients   map[string]*client.Client
}

var _ io.Closer = (*XClient)(nil)

func (xc *XClient) Close() error {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()

	for key, client := range xc.clients {
		client.Close()
		delete(xc.clients, key)
	}

	return nil
}

func NewXClient(d Discovery, mode SelectMode, opt *server.Option) *XClient {
	return &XClient{
		discovery: d,
		mode:      mode,
		opt:       opt,
		clients:   make(map[string]*client.Client),
	}
}

// call

func (xc *XClient) dial(rpcAddr string) (*client.Client, error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()

	cl, ok := xc.clients[rpcAddr]
	// client exist(cache) but doesn't work
	if ok && !cl.IsAvailable() {
		cl.Close()
		delete(xc.clients, rpcAddr)
		// reacquire
		cl = nil
	}

	// client doesn't exist, reacquire rpc
	if cl == nil {
		var err error
		cl, err = client.SDial(rpcAddr, xc.opt)
		if err != nil {
			return nil, err
		}
		// add client cache
		xc.clients[rpcAddr] = cl
	}

	return cl, nil
}

func (xc *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
	client, err := xc.dial(rpcAddr)
	if err != nil {
		return err
	}

	return client.Call(ctx, serviceMethod, args, reply)
}

func (xc *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	rpcAddr, err := xc.discovery.Get(xc.mode)
	if err != nil {
		return err
	}

	return xc.call(rpcAddr, ctx, serviceMethod, args, reply)
}

// Broadcast 将请求广播到所有的服务实例，
// 如果任意一个实例发生错误，则返回其中一个错误；如果调用成功，则返回其中一个的结果。
func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	servers, err := xc.discovery.GetAll()
	if err != nil {
		return err
	}

	// protect e and replyDone
	var mutex sync.Mutex
	var wg sync.WaitGroup
	var e error
	// if reply is nil, don't need to set value
	replyDone := reply == nil
	// if exist error, cancel left processes
	ctx, cancel := context.WithCancel(ctx)

	for _, rpcAddr := range servers {
		wg.Add(1)

		go func(rpcAddr string) {
			defer wg.Done()

			var cloneReply interface{}
			if reply != nil {
				cloneReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err := xc.call(rpcAddr, ctx, serviceMethod, args, reply)

			mutex.Lock()
			if err != nil && e == nil {
				e = err
				// if any call failed, cancel unfinished calls
				cancel()
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(cloneReply).Elem())
				replyDone = true
			}
			mutex.Unlock()
		}(rpcAddr)
	}

	wg.Wait()
	return e
}
