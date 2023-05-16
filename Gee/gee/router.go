package gee

import (
	"log"
	"net/http"
	"strings"
)

// 处理函数接口
type HandlerFunc func(*Context)

// roots key eg, roots['GET'] roots['POST']
// handlers key eg, handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']
type Router struct {
	roots	map[string]*node
	// 通过键值对查找对应的路由和HandleFunc
	handlers map[string]HandlerFunc
}

// 实例化路由器
func newRouter() *Router {
	return &Router{
		roots: make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 将pattern转换成string[]，即parts
func parsePattern(pattern string) []string {
	ps := strings.Split(pattern, "/")
	parts := make([]string, 0)

	for _, p := range ps {
		if p != "" {
			parts = append(parts, p)
			// 匹配到*，直接结束
			if(p[0] == '*') {
				break
			}
		}
	}

	return parts
}

// 添加路由
func (router *Router) addRoute(method string, pattern string, handler HandlerFunc) {
	log.Printf("Register Route %4s - %s", method, pattern)

	// 解析pattern
	parts := parsePattern(pattern)
	key := method + "-" + pattern

	// 如果roots[method]没实例化，则创建一个
	if _, ok := router.roots[method]; !ok {
		router.roots[method] = &node{}
	}

	// 插入到前缀树
	router.roots[method].insert(pattern, parts, 0)
	// 添加对应的handler
	router.handlers[key] = handler
}

// 获取路由
func (router *Router) getRoute(method string, path string) (*node, map[string]string) {
	// 参数Map
	params := make(map[string]string)
	// 实际路由的parts
	searchParts := parsePattern(path)
	root, ok := router.roots[method]

	if !ok {
		return nil, nil
	}

	// 查找节点
	node := root.search(searchParts, 0)
	if node == nil {
		return nil, nil
	}

	// 从节点的逻辑parts中找出params，即模糊查询的参数
	parts := parsePattern(node.pattern)
	for i, part := range parts {
		if part[0] == ':' {
			// 拿到:后面的模糊匹配字符
			params[part[1:]] = searchParts[i]
		} else if part[0] == '*' && len(part) > 1 {
			// 拿到*后面的模糊匹配字符， 将searchParts[i:]后的字符都是*的匹配字符
			params[part[1:]] = strings.Join(searchParts[i:], "/")
			break
		}
	}

	return node, params
}

// 路由处理
func (router *Router) handler(ctx *Context) {
	// 获取到指定路由
	route, params := router.getRoute(ctx.Method, ctx.Path)

	if route != nil {
		// 给ctx传入params
		ctx.Params = params
		key := ctx.Method + "-" + route.pattern

		// 将路由处理作为最后一个中间件去执行
		ctx.middlewares = append(ctx.middlewares, router.handlers[key])
	} else {
		ctx.middlewares = append(ctx.middlewares, func(ctx *Context) {
			ctx.String(http.StatusNotFound, "404 NOT FOUND: %s\n", ctx.Path)
		})
	}

	// 开始执行所有中间件
	ctx.Next()
}