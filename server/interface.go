package server

import (
	"local/sndaRpc/util"

	"github.com/go-kit/kit/log"
)

// Server 服务接口
type Server interface {
	// SetLogger 设置logger
	SetLogger(lg log.Logger) error
	RegisterByConfig(serverInfo *util.ServerInfo) error
	Register(handlerInterface, handlerCls interface{}, serviceName, protoName string, methodList ...*MethodInfo) error
	Serve(addr string) error
}
