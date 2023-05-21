package geecache

import (
	"fmt"
	"geecache/geecache/consistenthash"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

/* -------------------------- 提供缓存被其他节点访问的能力(基于http) -------------------------- */

const (
	defaultBasePath = "_geecache"
	defaultReplicas = 30
)

type HTTPPool struct {
	// 记录自己的http地址
	self string
	// 前缀
	basePath string
	// 保护peer和httpGetter
	mutex sync.Mutex
	// 存储所有分布式节点，并使用一致性哈希
	peers *consistenthash.Map
	// keyed by e.g. "http://10.0.0.2:8008"
	httpGetters map[string]*httpGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (pool *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", pool.self, fmt.Sprintf(format, v...))
}

// 实现http.Handler
func (pool *HTTPPool) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// 是否有缓存前缀
	if strings.HasPrefix(req.URL.Path, pool.basePath) {
		panic("HTTPPool serving unexpected path" + req.URL.Path)
	}

	pool.Log("%s %s", req.Method, req.URL.Path)

	// url的分类： /<basePath>/<group>/<key>
	// req.URL.Path[len(pool.basePath) + 2:]: 切片后拿到了 <group>/<key>，+2 是去除前后斜杠
	parts := strings.SplitN(req.URL.Path[len(pool.basePath)+2:], "/", 2)
	if len(parts) != 2 {
		http.Error(res, "invalid url", http.StatusBadRequest)
		return
	}

	// 拿到指定group
	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(res, "invalid group: "+groupName, http.StatusNotFound)
		return
	}

	// 拿到对应value
	bytes, err := group.Get(key)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// application/octet-stream: 字节流传输
	res.Header().Set("Content-Type", "application/octet-stream")
	res.Write(bytes.ByteSlice())
	// 字符串
	// res.Header().Set("Content-Type", "text/plain")
	// res.Write(bytes.ByteSlice())
}

/* ---------------------------------- peer ---------------------------------- */

type httpGetter struct {
	// 远程节点的地址
	baseUrl string
}

// 实现PeerGetter接口，获取远程节点上的缓存值
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 获取远程节点的地址
	// fmt.Println(fmt.Sprintf("%v%v/%v", h.baseUrl, url.QueryEscape(group), url.QueryEscape(key)))
	// fmt.Println(fmt.Sprintf(h.baseUrl, url.QueryEscape(group), url.QueryEscape(key)))
	// url.QueryEscape 字符转义
	remoteUrl := fmt.Sprintf("%v/%v/%v", h.baseUrl, url.QueryEscape(group), url.QueryEscape(key))

	// HTTP客户端请求获取缓存
	res, err := http.Get(remoteUrl)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("server returned: %v", res.Status)
	}

	// 拿到缓存值
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}

	log.Printf("[Peer %s] get key %s",h.baseUrl, key)

	return bytes, nil
}

// 实现PeerPicker接口
// 实例化了一致性哈希算法，并且添加了传入的节点
// peers为所有分布式节点的地址数组
func (pool *HTTPPool) Set(peers ...string) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	// 一致性哈希
	pool.peers = consistenthash.NewMap(defaultReplicas, nil)
	// 添加到哈希环中
	pool.peers.Add(peers...)
	// peer地址到HTTP客户端的映射
	pool.httpGetters = make(map[string]*httpGetter, len(peers))
	// 每一个节点创建了一个 HTTP 客户端
	for _, peer := range peers {
		pool.httpGetters[peer] = &httpGetter{baseUrl: peer + "/" + pool.basePath}
	}
}

// 根据具体的 key，选择节点，返回节点对应的 HTTP 客户端
func (pool *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	// 通过哈希环找到最近的远程节点
	// peer == pool.self 说明命中的是本地缓存服务器
	if peer := pool.peers.Get(key); peer != "" && peer != pool.self {
		pool.Log("Pick Peer %s", peer)
		return pool.httpGetters[peer], true
	}

	return nil, false
}
