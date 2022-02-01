package codec
import "io"

type Header struct{
	ServiceMethod string  `json:"ServiceMethod"`//"调用方法 格式 service.method"
	Seq uint64   `json:"Seq"`//客户端选择的序列号
	Error string `json:"Error"`
 }

//编码器是一个接口，需要实现:关闭数据流，读，写等方法

type Codec interface{
	io.Closer //关闭数据流的接口，即需要一个close函数
	ReadBody(interface{}) error
	ReadHeader(*Header) error
	Write(*Header,interface{}) error // 写-写头部和body

}

//codec 的构造函数,传入一个io.readwritecloser 实例，返回一个codec实例
type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

//不同的序列化方法
const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // not implemented
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init(){
	NewCodecFuncMap = make(map[Type]NewCodecFunc) //string-func 的map
	NewCodecFuncMap[GobType] = NewGobCodec //新建一个Gob编码器
	NewCodecFuncMap[JsonType] = NewJsonCodec //新建一个Json编码器
}