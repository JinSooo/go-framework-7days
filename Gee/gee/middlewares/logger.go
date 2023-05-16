package middlewares

import (
	"gee-demo/gee"
	"log"
	"time"
)

func Logger() gee.HandlerFunc {
	return func(ctx *gee.Context) {
		start := time.Now()
		ctx.Next()
		log.Printf("[M - Logger] [%d] %s in %v", ctx.StatusCode, ctx.Req.RequestURI, time.Since(start))
	}
}