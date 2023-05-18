package gee

import (
	"log"
	"time"
)

func Logger() HandlerFunc {
	return func(ctx *Context) {
		start := time.Now()
		ctx.Next()
		log.Printf("[M - Logger] [%d] %s in %v", ctx.StatusCode, ctx.Req.RequestURI, time.Since(start))
	}
}