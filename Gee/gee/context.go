package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// 给map[string]interface{}起了一个别名gee.H，构建JSON数据时，显得更简洁
type H map[string]interface{}

// 封装上下文
type Context struct {
	// origin
	Req *http.Request
	Res http.ResponseWriter
	// req
	Path string
	Method string
	Params map[string]string
	// res
	StatusCode int
	// middlewares
	middlewares []HandlerFunc
	// 当前middlewares的执行位置
	index int
	// engine
	engine *Engine
}

// 工厂函数，实例化一个Context
func newContext(res http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Req: req,
		Res: res,
		Path: req.URL.Path,
		Method: req.Method,
		index: -1,
	}
}

// 执行下一个中间件
func (ctx *Context) Next() {
	// size := len(ctx.middlewares)
	// ctx.index++
	// for ; ctx.index < size; ctx.index++  {
	// 	ctx.middlewares[ctx.index](ctx)
	// }
	ctx.index++
	// 执行接下来的中间件
	ctx.middlewares[ctx.index](ctx)
}

// 获取指定路由参数
func (ctx *Context) Param(key string) string {
	return ctx.Params[key]
}

// 获取表单属性
func (ctx *Context) PostForm(key string) string{
	return ctx.Req.FormValue(key)
}

// 查询Query值
func (ctx *Context) Query(key string) string{
	return ctx.Req.URL.Query().Get(key)
}

// 设置状态码
func (ctx *Context) Status(code int){
	ctx.StatusCode = code
	ctx.Res.WriteHeader(code)
}

// 设置响应头
func (ctx *Context) SetHeader(key string, value string){
	ctx.Res.Header().Set(key, value)
}

// 以String类型返回
func (ctx *Context) String(code int, format string, values ...interface{}) {
	// 注意调用顺序应该是Header().Set 然后WriteHeader() 最后是Write()，不然header不会生效
	ctx.SetHeader("Content-Type", "text/plain")
	ctx.Status(code)
	ctx.Res.Write([]byte(fmt.Sprintf(format, values...)))
}

// 以JSON类型返回
func (ctx *Context) JSON(code int, obj interface{}) {
	ctx.SetHeader("Content-Type", "application/json")
	ctx.Status(code)
	encoder := json.NewEncoder(ctx.Res)
	if err := encoder.Encode(obj); err != nil {
		http.Error(ctx.Res, err.Error(), 500)
	}
}

// 直接返回data
func (ctx *Context) Data(code int, data []byte) {
	ctx.Status(code)
	ctx.Res.Write(data)
}

// 返回HTML
func (ctx *Context) HTML(code int, templateName string, data interface{}) {
	ctx.SetHeader("Content-Type", "text/html")
	ctx.Status(code)
	// ctx.Res.Write([]byte(html))
	if err := ctx.engine.htmlTemplates.ExecuteTemplate(ctx.Res, templateName, data); err != nil {
		http.Error(ctx.Res, err.Error(), 500)
	}
}