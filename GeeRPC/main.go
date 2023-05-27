package main

import (
	"geerpc/client"
	"geerpc/server"
	"log"
	"net"
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
	if err := server.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}

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

			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
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
