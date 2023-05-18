package gee

import (
	"fmt"
	"net/http"
	"strings"
	"text/template"
)


type Engine struct {
	*RouterGroup
	router *Router
	groups []*RouterGroup

	// html渲染
	// 所有HTML模板
	htmlTemplates *template.Template
	// 自定义模板渲染函数，用于模板里的函数调用
	funcMap template.FuncMap
}

// 实例化一个Engine
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}

	// 添加默认中间件
	// 错误恢复
	engine.Use(Recovery())

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
	ctx.engine = engine
	engine.router.handler(ctx)
}

/* ------------------------------- HTML Render ------------------------------ */
// 设置自定义渲染函数
func (engine *Engine) SetFuncMap(funcMap template.FuncMap)  {
	engine.funcMap = funcMap
}

// 加载HTML模板
func (engine *Engine) LoadHTMLGlob(templatePath string)  {
	// 实例化一个模板并将Funcs加入进去，并执行解析的模板文件夹
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(templatePath))
}