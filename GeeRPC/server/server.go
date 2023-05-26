package server

import (
	"encoding/json"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

/* --------------------------------- 需要协商的内容 -------------------------------- */
/**
 * | Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
 * | <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   -------> |
 *
 * Option 固定在报文的最开始，Header 和 Body 可以有多个
 * | Option | Header1 | Body1 | Header2 | Body2 | ...
 */

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this is a geerpc request
	CodecType   codec.Type // client chooses the codec type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

/* --------------------------------- Server --------------------------------- */

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = &Server{}

// loop waiting for a socket connection
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		fmt.Println("get a conn")

		go server.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

// ServeConn runs the server on a single connection
// get option instance and process negotiation logic
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { conn.Close() }()

	// get option form client
	var option Option
	if err := json.NewDecoder(conn).Decode(&option); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if option.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", option.MagicNumber)
		return
	}

	fn := codec.NewCodecFuncMap[option.CodecType]
	if fn == nil {
		log.Printf("rpc server: invalid codec type %s", option.CodecType)
		return
	}

	server.serveCodec(fn(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	// make sure to send a complete response
	sending := new(sync.Mutex)
	// wait until all requests are handled
	wg := new(sync.WaitGroup)

	for {
		// read request
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			// send response
			// return error to client
			req.header.Error = err.Error()
			server.sendResponse(cc, req.header, invalidRequest, sending)
			return
		}

		// handle request
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg)
	}
}

/* --------------------------------- request -------------------------------- */

type request struct {
	header *codec.Header // header of request
	argv   reflect.Value
	replyv reflect.Value
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var header codec.Header

	if err := cc.ReadHeader(&header); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}

	return &header, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	header, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}

	req := &request{header: header}

	req.argv = reflect.New(reflect.TypeOf(""))
	if err := cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
		// return nil, err
	}

	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, header *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()

	if err := cc.Write(header, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	// TODO: should call registered rpc methods to get the right replyv
	log.Println("[server]", req.header, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.header.Seq))
	server.sendResponse(cc, req.header, req.replyv.Interface(), sending)
}
