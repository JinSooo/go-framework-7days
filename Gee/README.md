# Gee

> 2023-5-15 6:21PM

一个简单的 Web 框架

## 特性 👇👇👇

### 基本使用

```go
router := gee.Default()

router.Get("/", func(c *gee.Context) {
	c.String(http.StatusOK, "Hello Gee\n")
})

router.Run(":8080")
```

## 基本路由（GET、POST）

封装了常用的响应方式（String、JSON、HTML...）

```go
router.Get("/hello", func(ctx *gee.Context) {
		ctx.Data(http.StatusOK, []byte("hello"))
})

router.Get("/", func(ctx *gee.Context) {
	ctx.String(http.StatusOK, "URL.Path = %q\n", ctx.Path)
})

router.Get("/kv", func(ctx *gee.Context) {
	for k, v := range ctx.Req.Header {
		ctx.String(http.StatusOK, "Header[%q] = %q\n", k, v)
	}
})

router.Get("/query", func(ctx *gee.Context) {
	ctx.String(http.StatusOK, "Query %v", ctx.Query("t"))
})

router.Get("/json", func(ctx *gee.Context) {
	ctx.SetHeader("Prisma", "ok")
	obj := make(gee.H)
	obj["name"] = "123"
	obj["age"] = "456"
	ctx.JSON(http.StatusOK, obj)
})
```

## 上下文 Context 统一处理

将请求和响应封装到 Context 中，并且通过它们封装了一些常用的方法

```go
type Context struct {
	// origin
	Req *http.Request
	Res http.ResponseWriter
	// req
	Path   string
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
```

## 前缀树路由，可以进行模糊匹配（/\*，/:id）

```go
router.Get("/hello/:name", func(c *gee.Context) {
	// expect /hello/geektutu
	c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
})

router.Get("/assets/*filepath", func(c *gee.Context) {
	c.JSON(http.StatusOK, gee.H{"filepath": c.Param("filepath")})
})
```

## 中间件

gee 内部的默认实例方法，默认添加了两个中间件

```go
func Default() *Engine {
	engine := New()

	engine.Use(Logger(), Recovery())

	return engine
}
```

自定义中间件

```go
func onlyForV2() gee.HandlerFunc {
	return func(ctx *gee.Context) {
		start := time.Now()
		ctx.Next()
		log.Printf("[M - onlyForV2] [%d] %s in %v", ctx.StatusCode, ctx.Req.RequestURI, time.Since(start))
	}
}

router.Use(onlyForV2())
```

## 路由分组

```go
v1 := router.Group("/v1")
{
	v1.Get("/", func(c *gee.Context) {
		c.String(http.StatusOK, "Hello /v1\n")
	})
	v1.Get("/hello", func(c *gee.Context) {
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})
}

v2 := router.Group("/v2")
{
	v2.Get("/hello/:name", func(c *gee.Context) {
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
	})
	v2.Post("/login", func(c *gee.Context) {
		c.JSON(http.StatusOK, gee.H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})
}
```

## HTML 模板

```go
// 注意顺序
router.SetFuncMap(template.FuncMap{
	"FormatAsDate": FormatAsDate,
})
router.LoadHTMLGlob("templates/*")

stu1 := &student{Name: "Geektutu", Age: 20}
stu2 := &student{Name: "Jack", Age: 22}
router.Get("/", func(c *gee.Context) {
	c.HTML(http.StatusOK, "css.tmpl", gee.H{
		"title": "gee",
		"now":   time.Now(),
	})
})

router.Get("/students", func(c *gee.Context) {
	c.HTML(http.StatusOK, "arr.tmpl", gee.H{
		"title":  "gee",
		"stuArr": [2]*student{stu1, stu2},
	})
})

router.Get("/date", func(c *gee.Context) {
	c.HTML(http.StatusOK, "custom_func.tmpl", gee.H{
		"title": "gee",
		"now":   time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC),
	})
})
```

## 静态文件服务

```go
// Static(路由，文件目录)
router.Static("/assets", "./assets")
```

## 错误恢复

当访问`/panic`后，服务器会报数组越界的错误，在使用了 Recovery 中间件后，会自动恢复

```go
router.Get("/panic", func(c *gee.Context) {
	names := []string{"geektutu"}
	c.String(http.StatusOK, names[100])
})
```
