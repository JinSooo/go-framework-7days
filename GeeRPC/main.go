package main

import (
	"encoding/json"
	"fmt"
	"geerpc/codec"
	"geerpc/server"
	"log"
	"net"
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
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { conn.Close() }()

	time.Sleep(time.Second)

	_ = json.NewEncoder(conn).Encode(server.DefaultOption)
	cc := codec.NewGobCodec(conn)

	for i := 0; i < 5; i++ {
		header := &codec.Header{
			ServiceMethod: "RPC.Sum",
			Seq:           uint64(i),
		}
		_ = cc.Write(header, fmt.Sprintf("geerpc req %d", header.Seq))
		_ = cc.ReadHeader(header)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	startClient(addr)
}
