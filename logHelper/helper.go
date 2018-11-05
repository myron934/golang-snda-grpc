package logHelper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"local/sndaRpc/util"
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var (
	logMap map[string]log.Logger
)

func init() {
	logMap = make(map[string]log.Logger)
}

//logKey 记录日志用的key
type logKey struct {
}

//LogInfo 记录日志用的一些信息
type LogInfo struct {
	FlowID string
}

//FromContext 从context里面获取log相关的内容
func FromContext(ctx context.Context) (info *LogInfo, ok bool) {
	info, ok = ctx.Value(LogInfo{}).(*LogInfo)
	return
}

//NewContext 放入log信息到context中
func NewContext() context.Context {
	return ContextWithNewLogInfo(context.Background())
}

//ContextWithLogInfo 放入log信息到context中
func ContextWithLogInfo(ctx context.Context, info *LogInfo) context.Context {
	return context.WithValue(ctx, LogInfo{}, info)
}

//ContextWithNewLogInfo 放入log信息到context中
func ContextWithNewLogInfo(ctx context.Context) context.Context {
	var logInfo = LogInfo{FlowID: util.NewGuid()}
	return ContextWithLogInfo(ctx, &logInfo)
}

// 注册线程安全的日志
// name为log名字, 例如 all.
// fileName 为日志文件名
// logLevel为日志级别 取值有: error, warn, info, debug, all.
func Register(name string, fileName string, logLevel string) (log.Logger, error) {
	_, ok := logMap[name]
	if ok {
		return nil, errors.New("log" + name + "exist")
	}
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm|os.ModeAppend)
	if err != nil {
		return nil, err
	}
	w := log.NewSyncWriter(file)
	return RegisterByWriter(name, w, logLevel)
}

func RegisterByWriter(name string, w io.Writer, logLevel string) (log.Logger, error) {
	_, ok := logMap[name]
	if ok {
		return nil, fmt.Errorf("log %s exist already", name)
	}
	logger := log.NewLogfmtLogger(w)
	//logger := log.NewJSONLogger(w)
	logLevel = strings.ToLower(logLevel)
	switch logLevel {
	case "error":
		logger = level.NewFilter(logger, level.AllowError())
	case "warn":
		logger = level.NewFilter(logger, level.AllowWarn())
	case "info":
		logger = level.NewFilter(logger, level.AllowInfo())
	case "debug":
		logger = level.NewFilter(logger, level.AllowDebug())
	default:
		logger = level.NewFilter(logger, level.AllowAll())
	}
	logMap[name] = logger
	return logger, nil
}

func Logger(name string) log.Logger {
	return logMap[name]
}

func Debug(name string) log.Logger {
	return level.Debug(logMap[name])
}

func Error(name string) log.Logger {
	return level.Error(logMap[name])
}

func Info(name string) log.Logger {
	return level.Info(logMap[name])
}

func Warn(name string) log.Logger {
	return level.Warn(logMap[name])
}

//func Debug( name string, args ... interface{}) error{
//	logger,ok:=logMap[name]
//	if !ok{
//		return errors.New("log "+ name+" is no exist")
//	}
//	level.Debug(logger).Log(args)
//	return nil
//}
//
//func Error( name string, args ... interface{}) error{
//	logger,ok:=logMap[name]
//	if !ok{
//		return errors.New("log "+ name+" is no exist")
//	}
//	level.Error(logger).Log(args)
//	return nil
//}
//
//func Info( name string, args ... interface{}) error{
//	logger,ok:=logMap[name]
//	if !ok{
//		return errors.New("log "+ name+" is no exist")
//	}
//	level.Info(logger).Log(args)
//	return nil
//}
//
//
//func Warn( name string, args ... interface{}) error{
//	logger,ok:=logMap[name]
//	if !ok{
//		return errors.New("log "+ name+" is no exist")
//	}
//	level.Warn(logger).Log(args)
//	return nil
//}
