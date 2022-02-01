package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

//gob实现了codec接口

type GobCodec struct {
	conn  io.ReadWriteCloser
	enc *gob.Encoder
	dec *gob.Decoder
	buf *bufio.Writer
}
//GobCodec 的构造方法
func NewGobCodec(conn io.ReadWriteCloser) Codec{
	buff := bufio.NewWriter(conn)
	return &GobCodec{
		conn:conn,
		buf:buff,
		enc:gob.NewEncoder(buff), //编码到buff里
		dec:gob.NewDecoder(conn),	//从conn解码
	}
}

//实现GobCodec的读写方法

func (c * GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}
func (c *GobCodec) ReadBody(body interface{}) error{
	return c.dec.Decode(body)
}
func (c * GobCodec) Write(h *Header,body interface{}) (err error){
	// 写完要从buf里flush到io然后关闭
	defer func(){
		_ =c.buf.Flush()
		if err !=nil{
			_ = c.Close()
		}
	}()

	if err:= c.enc.Encode(h);err !=nil{
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {

		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return  nil
}
func (c *GobCodec) Close() error{
	return c.conn.Close()

}