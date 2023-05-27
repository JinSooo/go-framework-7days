# GeeORM

> start: 2023-5-25 10:45PM
>
> end: 2023-5-27 11:44PM

RPC(Remote Procedure Call，远程过程调用)是一种计算机通信协议，允许调用不同进程空间的程序。RPC 的客户端和服务器可以在一台机器上，也可以在不同的机器上。程序员使用时，就像调用本地程序一样，无需关注内部的实现细节。

GeeRPC 选择从零实现 Go 语言官方的标准库 net/rpc，并在此基础上，新增了协议交换(protocol exchange)、注册中心(registry)、服务发现(service discovery)、负载均衡(load balance)、超时处理(timeout processing)等特性。

## reflect 相关知识

### reflect 获取方法参数

```go
var wg sync.WaitGroup
typ := reflect.TypeOf(&wg)
for i := 0; i < typ.NumMethod(); i++ {
	method := typ.Method(i)
	args := make([]string, 0, method.Type.NumIn())
	returns := make([]string, 0, method.Type.NumOut())
	for j := 1; j < method.Type.NumIn(); j++ {
		args = append(args, method.Type.In(j).Name())
	}
	for j := 0; j < method.Type.NumOut(); j++ {
		returns = append(returns, method.Type.Out(j).Name())
	}
	log.Printf("func (w *%s) %s(%s) %s",
		typ.Elem().Name(),
		method.Name,
		strings.Join(args, ","),
		strings.Join(returns, ","))
}
```

```
2023/05/27 11:43:00 func (w *WaitGroup) Add(int)
2023/05/27 11:43:00 func (w *WaitGroup) Done()
2023/05/27 11:43:00 func (w \*WaitGroup) Wait()
```

## 基本使用

需要导出的方法

```go
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
```

```go
package main

import (
	"context"
	"geerpc/registry"
	"geerpc/server"
	"geerpc/xclient"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// 注册中心
func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP()
	wg.Done()
	_ = http.Serve(l, nil)
}

// 服务器
func startServer(registryAddr string, wg *sync.WaitGroup) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")
	server := server.NewServer()
	_ = server.Register(&foo)
	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	wg.Done()
	server.Accept(l)
}

// RPC方法调用封装
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

// RPC方法调用
func call(registry string) {
	d := xclient.NewGeeRegistryDiscovery(registry, 0)
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

func main() {
	log.SetFlags(0)
	registryAddr := "http://localhost:9999/_geerpc/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	wg.Add(2)
	go startServer(registryAddr, &wg)
	go startServer(registryAddr, &wg)
	wg.Wait()

	time.Sleep(time.Second)
	call(registryAddr)
}

```

# 特性

## 消息编码

```
| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   -------> |
Option 固定在报文的最开始，Header 和 Body 可以有多个
| Option | Header1 | Body1 | Header2 | Body2 | ...
```

## 服务注册

### RPC 方法需要满足的条件

- the method’s type is exported. – 方法所属类型是导出的。
- the method is exported. – 方式是导出的。
- the method has two arguments, both exported (or builtin) types. – 两个入参，均为导出或内置类型。
- the method’s second argument is a pointer. – 第二个入参必须是一个指针。
- the method has return type error. – 返回值为 error 类型。

将其注册为一个 RPC 方法

```go
func (t *T) MethodName(argType T1, replyType *T2) error
```

这里就需要用到上面的 reflect

### reflect 获取方法小 Demo

```go
func main() {
	var wg sync.WaitGroup
	typ := reflect.TypeOf(&wg)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		argv := make([]string, 0, method.Type.NumIn())
		returns := make([]string, 0, method.Type.NumOut())
		// j 从 1 开始，第 0 个入参是 wg 自己。
		for j := 1; j < method.Type.NumIn(); j++ {
			argv = append(argv, method.Type.In(j).Name())
		}
		for j := 0; j < method.Type.NumOut(); j++ {
			returns = append(returns, method.Type.Out(j).Name())
		}
		log.Printf("func (w *%s) %s(%s) %s",
			typ.Elem().Name(),
			method.Name,
			strings.Join(argv, ","),
			strings.Join(returns, ","))
    }
}
```

结果:

```
func (w *WaitGroup) Add(int)
func (w *WaitGroup) Done()
func (w *WaitGroup) Wait()
```

## 超时处理

纵观整个远程调用的过程，需要客户端处理超时的地方有：

与服务端建立连接，导致的超时
发送请求到服务端，写报文导致的超时
等待服务端处理时，等待处理导致的超时（比如服务端已挂死，迟迟不响应）
从服务端接收响应时，读报文导致的超时
需要服务端处理超时的地方有：

读取客户端请求报文时，读报文导致的超时
发送响应报文时，写报文导致的超时
调用映射服务的方法时，处理报文导致的超时

有关一些处理的使用

1. 将调用放入协程中，同时设置一个管道来接收它，并且还有一个超时

```go
ch := make(chan int)
go func() {
	// ...
  ch <- 0
}()

select {
case <-time.After(opt.ConnectTimeout):
	return ...
case result := <-ch:
	return ...
}
```

2. 使用 Context 包实现超时处理，将控制器交给用户

```go
ctx, _ := context.WithTimeout(context.Background(), time.Second)
var reply int
err := client.Call(ctx, "Foo.Sum", &Args{1, 2}, &reply)
...

func (client *Client) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
}
```

## 支持 HTTP 协议

```go
func startServer(addrCh chan string) {
	var foo Foo
	l, _ := net.Listen("tcp", ":9999")
	_ = server.Register(&foo)
	server.HandleHTTP()
	addrCh <- l.Addr().String()
	_ = http.Serve(l, nil)
}
```

同时，你还可以访问 [http://localhost:9999/\_geerpc/debug](http://localhost:9999/_geerpc/debug) 查看 rpc 方法调用的次数

## 负载均衡

支持的负载均衡策略:

- 随机选择策略 - 从服务列表中随机选择一个。
- 轮询算法(Round Robin) - 依次调度不同的服务器，每次调度执行 i = (i + 1) mode n。

## 注册中心

![](https://geektutu.com/post/geerpc-day7/registry.jpg)
注册中心的位置如上图所示。注册中心的好处在于，客户端和服务端都只需要感知注册中心的存在，而无需感知对方的存在。更具体一些：

服务端启动后，向注册中心发送注册消息，注册中心得知该服务已经启动，处于可用状态。一般来说，服务端还需要定期向注册中心发送心跳，证明自己还活着。
客户端向注册中心询问，当前哪天服务是可用的，注册中心将可用的服务列表返回客户端。
客户端根据注册中心得到的服务列表，选择其中一个发起调用。

```go
func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP()
	wg.Done()
	_ = http.Serve(l, nil)
}
```
