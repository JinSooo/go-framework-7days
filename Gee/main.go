package main

import (
	"gee-demo/gee"
	"net/http"
)

func main() {
	router := gee.Default()

	router.Get("/", func(c *gee.Context) {
		c.String(http.StatusOK, "Hello Gee\n")
	})

	router.Run(":8080")
}
