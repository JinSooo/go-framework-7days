package main

import (
	"context"
	"geerpc/client"
	"geerpc/server"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addr chan<- string) {
	var foo Foo
	_ = server.Register(&foo)
	lis, _ := net.Listen("tcp", ":8080")
	server.HandleHTTP()
	addr <- lis.Addr().String()
	// server.Accept(lis)
	http.Serve(lis, nil)
}

func startClient(addr <-chan string) {
	client, _ := client.DialHTTP("tcp", <-addr)
	defer func() { client.Close() }()

	time.Sleep(time.Second)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call(context.Background(), "Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Printf("[reply] %d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	startClient(addr)

}
