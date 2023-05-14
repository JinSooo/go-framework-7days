package main

import (
	"fmt"
	"gee-demo/gee"
	"net/http"
)

func main() {
	router := gee.New()

	router.Get("/hello", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("Hello"))
	})
	router.Get("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
	})
	router.Get("/kv", func(w http.ResponseWriter, req *http.Request) {
		for k, v := range req.Header {
			fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
		}
	})

	router.Run(":8080")
}
