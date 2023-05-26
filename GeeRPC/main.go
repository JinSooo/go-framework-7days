package main

import (
	"fmt"
	"geerpc/client"
	"geerpc/server"
	"log"
	"net"
	"sync"
	"time"
)

func startServer(addr chan<- string) {
	// pick a free prot
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error: ", err)
	}

	log.Println("start rpc server on", lis.Addr())
	addr <- lis.Addr().String()
	server.Accept(lis)
}

func startClient(addr <-chan string) {
	client, _ := client.Dial("tcp", <-addr)
	defer func() { client.Close() }()

	time.Sleep(time.Second)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			if err := client.Call("Service.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	startClient(addr)
}
