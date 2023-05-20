package geecache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

/* -------------------------- 提供缓存被其他节点访问的能力(基于http) -------------------------- */

const defaultBasePath = "_geecache"

type HTTPPool struct {
	// 记录自己的http地址
	self string
	// 前缀
	basePath string
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
	parts := strings.SplitN(req.URL.Path[len(pool.basePath) + 2:], "/", 2)
	if len(parts) != 2 {
		http.Error(res, "invalid url", http.StatusBadRequest)
		return
	}

	// 拿到指定group
	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(res, "invalid group: " + groupName, http.StatusNotFound)
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
