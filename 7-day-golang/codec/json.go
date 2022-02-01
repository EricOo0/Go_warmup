package codec

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

type JsonCodec struct {
	conn  io.ReadWriteCloser
	enc *json.Encoder
	dec * json.Decoder
	buf *bufio.Writer
}
func NewJsonCodec(conn io.ReadWriteCloser) Codec{
	buff := bufio.NewWriter(conn)
	return &JsonCodec{
		conn:conn,
		buf:buff,
		enc:json.NewEncoder(buff), //编码到buff里
		dec:json.NewDecoder(conn),	//从conn解码
	}
}

//实现JsonCodec的读写方法
func (c * JsonCodec) ReadHeader(h *Header) error {


	err :=c.dec.Decode(h)

	 return err
}
func (c *JsonCodec) ReadBody(body interface{}) error{
	return c.dec.Decode(body)
}
func (c * JsonCodec) Write(h *Header,body interface{}) (err error){
	// 写完要从buf里flush到io然后关闭
	defer func(){

		_ =c.buf.Flush()
		if err !=nil{
			_ = c.Close()
		}

	}()
	if err:= c.enc.Encode(h);err !=nil{
		log.Println("rpc codec: json error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: json error encoding body:", err)
		return err
	}
	return  nil
}
func (c *JsonCodec) Close() error{
	return c.conn.Close()

}