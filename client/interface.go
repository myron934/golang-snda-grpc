package client

import (
	"context"
	"local/sndaRpc/util"
	"time"

	"github.com/go-kit/kit/log"
)

// Client 客户端接口
type Client interface {
	SetLogger(lg log.Logger) error
	Register(clientInfo *util.ClientInfo) error
	Invoke(ctx context.Context, method string, request interface{}) (response interface{}, err error)
	InvokeTimeout(ctx context.Context, method string, request interface{}, duration time.Duration) (response interface{}, err error)
	InterfaceInfo(name string) *util.InterfaceInfo
}
