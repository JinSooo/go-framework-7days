package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

/* -------------------------------- Codec gob ------------------------------- */

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	/**
	 * bufio.NewWriter(conn), gob.NewDecoder(conn), gob.NewEncoder(buf)注意这三个方法
	 * 	bufio.NewWriter(conn): 为conn创建一个缓冲区
	 * 	gob.NewDecoder(conn): 反序列化，注意这边是conn，也就是说，解码后数据存放在conn中
	 * 	gob.NewEncoder(buf): 序列化，注意这边是buf，也就是说，编码后数据存放在buf中
	 *		下文中还有一个方法: buf.Flush()，将buf的数据写入到conn
	 */

	buf := bufio.NewWriter(conn)

	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

func (codec *GobCodec) ReadHeader(header *Header) error {
	return codec.dec.Decode(header)
}

func (codec *GobCodec) ReadBody(body interface{}) error {
	return codec.dec.Decode(body)
}

func (codec *GobCodec) Write(header *Header, body interface{}) (err error) {
	defer func() {
		// write the data in buf to io.Writer(conn)
		_ = codec.buf.Flush()
		if err != nil {
			_ = codec.Close()
		}
	}()

	if err := codec.enc.Encode(header); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err := codec.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}

	return nil
}

func (codec *GobCodec) Close() error {
	return codec.conn.Close()
}
