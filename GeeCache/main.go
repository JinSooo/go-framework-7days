package main

import (
	"flag"
	"fmt"
	"geecache/geecache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 1
func testHTTP() {
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

// 2
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

		// res.Header().Set("Content-Type", "application/octet-stream")
		res.Header().Set("Content-Type", "text/plain")
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

func main() {
	testPeer()
}
