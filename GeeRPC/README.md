# GeeORM

> start: 2023-5-25 10:45PM

RPC(Remote Procedure Call，远程过程调用)是一种计算机通信协议，允许调用不同进程空间的程序。RPC 的客户端和服务器可以在一台机器上，也可以在不同的机器上。程序员使用时，就像调用本地程序一样，无需关注内部的实现细节。

GeeRPC 选择从零实现 Go 语言官方的标准库 net/rpc，并在此基础上，新增了协议交换(protocol exchange)、注册中心(registry)、服务发现(service discovery)、负载均衡(load balance)、超时处理(timeout processing)等特性。

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

2023/05/27 11:43:00 func (w *WaitGroup) Add(int)
2023/05/27 11:43:00 func (w *WaitGroup) Done()
2023/05/27 11:43:00 func (w \*WaitGroup) Wait()

```

```
