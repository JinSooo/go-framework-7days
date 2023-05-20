package main

import (
	"fmt"
	"geecache/geecache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(func(key string) ([]byte, error) {
		if key == "" {
			return []byte{}, fmt.Errorf("[Getter] invalid key")
		}

		fmt.Println("[SlowDB] search key", key)
		if value, ok := db[key]; ok {
			return []byte(value), nil
		}

		return nil, fmt.Errorf("%s not exist", key)
	}))

	addr := ":8080"
	fmt.Println("geecache is serving at ", addr)
	pool := geecache.NewHTTPPool(addr)
	log.Fatal(http.ListenAndServe(addr, pool))
}
