package gateway

import (
	"local/sndaRpc/util"

	"github.com/go-kit/kit/log"
)

//GateWay 网关接口
type GateWay interface {
	SetLogger(lg log.Logger) error
	Register(infoList []*util.HTTPGateWayInfo) error
	Serve(addr string) error
}
