package core

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

func getCore(filename string,level zapcore.LevelEnabler) zapcore.Core{
	lumberJackLogger:=&lumberjack.Logger{
		Filename: filename,
		MaxSize: 1,//在进行切割之前，日志文件的最大大小（以MB为单位）
		MaxBackups:1,//旧文件的个数
		MaxAge: 1, //天数
		Compress: false,
	}
	writer :=zapcore.AddSync(lumberJackLogger)
	//自定义时间格式
	config := zapcore.EncoderConfig{
		MessageKey:   "msg",  //结构化（json）输出：msg的key
		LevelKey:     "level",//结构化（json）输出：日志级别的key（INFO，WARN，ERROR等）
		TimeKey:      "ts",   //结构化（json）输出：时间的key（INFO，WARN，ERROR等）
		CallerKey:    "file", //结构化（json）输出：打印日志的文件对应的Key
		EncodeLevel:  zapcore.CapitalLevelEncoder, //将日志级别转换成大写（INFO，WARN，ERROR等）
		EncodeCaller: zapcore.ShortCallerEncoder, //采用短文件路径编码输出（test/main.go:14 ）
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},//输出的时间格式
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},//

	}
	encoder := zapcore.NewJSONEncoder(config)//创建编码格式

	core := zapcore.NewCore(encoder,writer,level)//创建一个core，日志格式
	return core
}
func Zap() (logger *zap.Logger){
	//0、判断文件夹是否存在，不存在就新建
	_, err := os.Stat("./Log")
	if err != nil{
		if os.IsNotExist(err) {
			//Log文件夹不存在，新建
			_ = os.Mkdir("./Log", os.ModePerm)
		}
	}

	//1、生成一个log对象，用于打印日志,日志分为4种等级debug，info，warn，error
	debugLog :=zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zap.DebugLevel
	})
	infoLog :=zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zap.InfoLevel
	})
	warnLog :=zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zap.WarnLevel
	})
	errorLog :=zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zap.ErrorLevel //error 和以上的(fatal)错误信息都打印到error里
	})

	cores :=[...]zapcore.Core{
		getCore("./Log/server_debug.log",debugLog),
		getCore("./Log/server_info.log",infoLog),
		getCore("./Log/server_warn.log",warnLog),
		getCore("./Log/server_error.log",errorLog),
	}
	return zap.New(zapcore.NewTee(cores[:]...),zap.AddCaller())//...表示变长参数，取决于cores数组有多长
}