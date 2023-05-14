package gee

import (
	"log"
	"net/http"
	"strings"
)

// 处理函数接口
type HandlerFunc func(*Context)

type Router struct {
	// 通过键值对查找对应的路由和HandleFunc
	handlers map[string]HandlerFunc
}

// 实例化路由器
func newRouter() *Router {
	return &Router{
		handlers: make(map[string]HandlerFunc),
	}
}

// 添加路由
func (router *Router) addRoute(method string, pattern string, handler HandlerFunc) {
	log.Printf("Route %4s - %s", method, pattern)

	key := strings.ToUpper(method) + "-" + pattern
	router.handlers[key] = handler
}

// 路由处理
func (router *Router) handler(ctx *Context) {
	key := ctx.Method + "-" + ctx.Path

	if handler, ok := router.handlers[key]; ok {
		handler(ctx)
	} else {
		ctx.String(http.StatusNotFound, "404 NOT FOUND: %s\n", ctx.Path)
	}
}