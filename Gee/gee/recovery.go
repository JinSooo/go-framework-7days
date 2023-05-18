package gee

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

// 追踪报错的堆栈位置
func trace(message string) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:]) // skip first 3 caller

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}

// 错误恢复
func Recovery() HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				log.Printf("[E - Panic] [500] %s\n\n", trace(message))
				ctx.Fatal(http.StatusInternalServerError, "Internal Server Error")
			}
		}()

		ctx.Next()
	}
}