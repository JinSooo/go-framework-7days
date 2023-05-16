package gee

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