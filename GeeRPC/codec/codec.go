package codec

import "io"

/* ------------------------------- 消息的序列化与反序列化 ------------------------------ */
/**
 * 典型 RPC 调用
 * 	err = client.Call("Arith.Multiply", args, &reply)
 *
 * 将请求和响应中的参数和返回值抽象为 body，剩余的信息放在 header 中
 */

type Header struct {
	ServiceMethod string // "Service.Method"
	Seq           uint64  // request sequence
	Error         string // error info from server
}

type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec
type Type string

// supported codec types
const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
