package main

import (
	"gee-demo/gee"
	"net/http"
)

func main() {
	router := gee.New()

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

	router.Run(":8080")
}
