package main

import (
	"gee-demo/gee"
	"net/http"
)

// test route1
func route1(router *gee.Engine) {
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
}

func route2(router *gee.Engine) {
	router.Get("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
	})

	router.Get("/hello", func(c *gee.Context) {
		// expect /hello?name=geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	router.Get("/hello/:name", func(c *gee.Context) {
		// expect /hello/geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
	})

	router.Get("/assets/*filepath", func(c *gee.Context) {
		c.JSON(http.StatusOK, gee.H{"filepath": c.Param("filepath")})
	})
}

func route3(router *gee.Engine) {
	router.Get("/index", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<h1>Index Page</h1>")
	})

	v1 := router.Group("/v1")
	{
		v1.Get("/", func(c *gee.Context) {
			c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
		})

		v1.Get("/hello", func(c *gee.Context) {
			// expect /hello?name=geektutu
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
		})
	}

	v2 := router.Group("/v2")
	{
		v2.Get("/hello/:name", func(c *gee.Context) {
			// expect /hello/geektutu
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
		})
		v2.Post("/login", func(c *gee.Context) {
			c.JSON(http.StatusOK, gee.H{
				"username": c.PostForm("username"),
				"password": c.PostForm("password"),
			})
		})

	}

}

func main() {
	router := gee.New()

	route3(router)

	router.Run(":8080")
}
