package gee

import (
	"fmt"
	"net/http"
	"strings"
)

// 处理函数接口
type HandleFunc func(http.ResponseWriter, *http.Request)

/**
 * 核心
 */
type Engine struct {
	// 通过键值对查找对应的路由和HandleFunc
	router map[string]HandleFunc
}

// 工厂函数，实例化一个Engine
func New() *Engine {
	return &Engine{router: make(map[string]HandleFunc)}
}

// 添加路由
func (engine *Engine) addRoute(method string, pattern string, handler HandleFunc) {
	key := strings.ToUpper(method) + "-" + pattern
	engine.router[key] = handler
}

func (engine *Engine) Get(pattern string, handler HandleFunc) {
	engine.addRoute("GET", pattern, handler)
}

func (engine *Engine) Post(pattern string, handler HandleFunc) {
	engine.addRoute("POST", pattern, handler)
}

// 开启一个http服务器，并传入engine实例实现的接口方法ServeHTTP
func (engine *Engine) Run(addr string) error {
	fmt.Printf("Server is running at http://127.0.0.1%v\n", addr)
	return http.ListenAndServe(addr, engine)
}

// 真正的处理请求的地方
func (engine *Engine) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// 当有请求进入时，取到对应的key
	key := req.Method + "-" + req.URL.Path

	if handler, ok := engine.router[key]; ok {
		handler(res, req)
	} else {
		fmt.Fprintf(res, "404 NOT FOUND: %s\n", req.URL)
	}
}
