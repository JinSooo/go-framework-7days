package gee

import (
	"fmt"
	"net/http"
	"strings"
)


type Engine struct {
	*RouterGroup
	router *Router
	groups []*RouterGroup
}

// 实例化一个Engine
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
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
	// 判断哪些中间件需要被执行
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		// 筛选对应的中间件
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}

	// 当来请求时，实例化一个Context
	ctx := newContext(res, req)
	ctx.middlewares = middlewares
	engine.router.handler(ctx)
}
