package gee

import (
	"fmt"
	"net/http"
)


type Engine struct {
	router *Router
}

// 实例化一个Engine
func New() *Engine {
	return &Engine{router: newRouter()}
}

func (engine *Engine) Get(pattern string, handler HandlerFunc) {
	engine.router.addRoute("GET", pattern, handler)
}

func (engine *Engine) Post(pattern string, handler HandlerFunc) {
	engine.router.addRoute("POST", pattern, handler)
}

// 开启一个http服务器，并传入engine实例实现的接口方法ServeHTTP
func (engine *Engine) Run(addr string) error {
	fmt.Printf("Server is running at http://127.0.0.1%v\n", addr)
	return http.ListenAndServe(addr, engine)
}

// 真正的处理请求的地方
func (engine *Engine) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// 当来请求时，实例化一个Context
	ctx := newContext(res, req)
	engine.router.handler(ctx)
}
