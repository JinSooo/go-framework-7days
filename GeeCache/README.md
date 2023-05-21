# GeeCache

> start: 2023-5-19 12:58AM

> end: 2023-5-21 08:05PM

一个简单的 分布式缓存 框架(groupcache)

设计一个分布式缓存系统，需要考虑资源控制、淘汰策略、并发、分布式节点通信等各个方面的问题。而且，针对不同的应用场景，还需要在不同的特性之间权衡，例如，是否需要支持缓存更新？还是假定缓存在淘汰之前是不允许改变的。不同的权衡对应着不同的实现。

groupcache 是 Go 语言版的 memcached，目的是在某些特定场合替代 memcached。groupcache 的作者也是 memcached 的作者。无论是了解单机缓存还是分布式缓存，深入学习这个库的实现都是非常有意义的。

GeeCache 基本上模仿了 groupcache 的实现。支持特性有：

- 单机缓存和基于 HTTP 的分布式缓存
- 最近最少访问(Least Recently Used, LRU) 缓存策略
- 使用 Go 锁机制防止缓存击穿
- 使用一致性哈希选择节点，实现负载均衡
- 使用 protobuf 优化节点间二进制通信
- ...

## 特性 👇👇👇

### 基本使用

```go
// 模拟数据库
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func testHTTP() {
  // NewGroup(gourpName, 内存大小, 源数据获取方法)
	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(func(key string) ([]byte, error) {
		if key == "" {
			return []byte{}, fmt.Errorf("[Getter] invalid key")
		}

		fmt.Println("[SlowDB] search key", key)
		if value, ok := db[key]; ok {
			return []byte(value), nil
		}

		return nil, fmt.Errorf("%s not exist", key)
	}))

	addr := ":8080"
	fmt.Println("geecache is serving at ", addr)
	pool := geecache.NewHTTPPool(addr)
	log.Fatal(http.ListenAndServe(addr, pool))
}
```

### 分布式使用

```go
// 创建一个group
func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(func(key string) ([]byte, error) {
		if key == "" {
			return []byte{}, fmt.Errorf("[Getter] invalid key")
		}

		fmt.Println("[SlowDB] search key", key)
		if value, ok := db[key]; ok {
			return []byte(value), nil
		}

		return nil, fmt.Errorf("%s not exist", key)
	}))
}

// 启动缓存服务器：创建 HTTPPool，添加节点信息，注册到 gee 中，启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知
func startCacheServer(addr string, addrs []string, group *geecache.Group) {
	peers := geecache.NewHTTPPool(addr)
	peers.Set(addrs...)
	group.RegisterPeers(peers)

	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 启动一个 API 服务（端口 9999），与用户进行交互，用户感知
func startAPIServer(addr string, group *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		key := req.URL.Query().Get("key")
		// 获取键值，3种方式，通过缓存、远程节点、本地节点
		view, err := group.Get(key)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "application/octet-stream")
		res.Write(view.ByteSlice())
	}))

	log.Println("fontend server is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], nil))
}

func testPeer() {
	// 命令行传入 port 和 api 2 个参数，用来在指定端口启动 HTTP 服务
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go startAPIServer(apiAddr, gee)
	}
	startCacheServer(addrMap[port], []string(addrs), gee)
}
```

运行远程节点和 api 服务节点

```bash
#!/bin/bash
# trap 命令用于在 shell 脚本退出时，删掉临时文件，结束子进程
trap "rm main;kill 0" EXIT

go build -o main
./main -port=8001 &
./main -port=8002 &
# 8003 cache 和9999 api两个服务是在一个里面的，让8003的缓存充当api服务的本地缓存服务器
./main -port=8003 -api=1 &

read -n 1
```

## LRU 缓存淘汰机制

最近最少使用，相对于仅考虑时间因素的 FIFO 和仅考虑访问频率的 LFU，LRU 算法可以认为是相对平衡的一种淘汰算法。LRU 认为，如果数据最近被访问过，那么将来被访问的概率也会更高。LRU 算法的实现非常简单，维护一个队列，如果某条记录被访问了，则移动到队尾，那么队首则是最近最少访问的数据，淘汰该条记录即可。

## 并发缓存

通过`Sync.Mutex`实现协程之间的互斥操作

## 缓存值的存储和获取流程

```
                           是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
```

## 远程节点 Peer

### 一致性哈希

这里就利用 `一致性哈希算法` ，将所有远程节点加入到哈希环中，再通过 key 的方式来查找（顺时针）最近的远程节点
![一致性哈希](https://geektutu.com/post/geecache-day4/add_peer.jpg)

#### 数据倾斜问题

通过将一个真实节点对于多个虚拟节点，虚拟节点映射到哈希环中，来扩充节点的数量

### 远程节点接口

`http://远程节点地址/_geecache/<groupname>/<key>`

### 分布式节点

```
使用一致性哈希选择节点       是                                    是
 	|-----> 是否是远程节点 -----> HTTP 客户端访问远程节点 --> 成功？-----> 服务端返回返回值
                 |  否                                    ↓  否
                 |----------------------------> 回退到本地节点处理。
```

## 防止缓存击穿

> 缓存雪崩：缓存在同一时刻全部失效，造成瞬时 DB 请求量大、压力骤增，引起雪崩。缓存雪崩通常因为缓存服务器宕机、缓存的 key 设置了相同的过期时间等引起。

> 缓存击穿：一个存在的 key，在缓存过期的一刻，同时有大量的请求，这些请求都会击穿到 DB ，造成瞬时 DB 请求量大、压力骤增。

> 缓存穿透：查询一个不存在的数据，因为不存在则不会写到缓存中，所以每次都会去请求 DB，如果瞬间流量过大，穿透到 DB，导致宕机。

使用 `singleflight` 防止缓存击穿

在并发的场景下，并发的请求了 N 个相同的 key，这是假设缓存服务器上没有缓存或缓存过期，那么这 N 个请求会引起服务器同时向数据库请求 N 次，容易导致缓存击穿和穿透。而且 HTTP 请求也是非常耗时的。

这时，我们就需要 `singleflight` 来将并发发起相同 key 的请求的进行合并，只进行一次请求，让其他相同的请求进行等待即可。

## Protobuf 通信

> protobuf 即 Protocol Buffers，Google 开发的一种数据描述语言，是一种轻便高效的结构化数据存储格式，与语言、平台无关，可扩展可序列化。protobuf 以二进制方式存储，占用空间小。

protobuf 广泛地应用于远程过程调用(RPC) 的二进制传输，使用 protobuf 的目的非常简单，为了获得更高的性能。传输前使用 protobuf 编码，接收方再进行解码，可以显著地降低二进制传输的大小。另外一方面，protobuf 可非常适合传输结构化数据，便于通信字段的扩展。

使用 protobuf 库，优化了节点间通信的性能。
