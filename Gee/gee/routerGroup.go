package gee

import (
	"net/http"
	"path"
)

type RouterGroup struct {
	// 父级Group，支持嵌套(我们设置根group为'/'，在engine中)
	parent *RouterGroup
	// group标识，路由前缀
	prefix string
	// engine实例
	engine *Engine
	// 中间件
	middlewares []HandlerFunc
}

// 创建一个新的路由分组
func (routerGroup *RouterGroup) Group(prefix string) *RouterGroup {
	engine := routerGroup.engine
	newGroup := &RouterGroup{
		prefix: routerGroup.prefix + prefix,
		parent: routerGroup,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)

	return newGroup
}

// 添加中间件
func (routerGroup *RouterGroup) Use(middlewares ...HandlerFunc) {
	routerGroup.middlewares = append(routerGroup.middlewares, middlewares...)
}

// 分组上添加路由
func (routerGroup *RouterGroup) addRoute(method string, pattern string, handler HandlerFunc) {
	compositionPattern := routerGroup.prefix + pattern

	// log.Printf("Register Route %4s - %s", method, compositionPattern)

	routerGroup.engine.router.addRoute(method, compositionPattern, handler)
}

func (routerGroup *RouterGroup) Get(pattern string, handler HandlerFunc) {
	routerGroup.addRoute("GET", pattern, handler)
}

func (routerGroup *RouterGroup) Post(pattern string, handler HandlerFunc) {
	routerGroup.addRoute("POST", pattern, handler)
}

/* ----------------------------- Static Resource ---------------------------- */
// 创建一个静态服务，将磁盘上的某个文件夹filePath映射到路由relativePath
func (routerGroup *RouterGroup) Static(relativePath string, filePath string) {
	pattern := path.Join(relativePath, "/*filepath")
	handler := routerGroup.createStaticHandler(relativePath, http.Dir(filePath))
	// 注册到路由中
	routerGroup.Get(pattern, handler)
}

// 静态文件服务的handler，处理对应的文件路由
func (routerGroup *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(routerGroup.prefix, relativePath)
	// StripPrefix返回一个处理程序，该处理程序通过从请求URL的路径(如果设置了RawPath)中删除给定的前缀并调用处理程序h来服务HTTP请求。
	// http.FileServer返回一个处理程序，该处理程序使用根文件系统的内容为HTTP请求提供服务。
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))

	return func(ctx *Context) {
		file := ctx.Param("filepath")

		// 文件是否存在
		if _, err := fs.Open(file); err != nil {
			ctx.Status(http.StatusNotFound)
			return
		}

		fileServer.ServeHTTP(ctx.Res, ctx.Req)
	}
}
