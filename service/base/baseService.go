package base

import (
	"errors"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/gogo/protobuf/proto"
	context "golang.org/x/net/context"
)

var (
	handlers map[string]grpctransport.Handler
)

func init() {
	handlers = make(map[string]grpctransport.Handler)
}

type BaseService struct {
}

func (base *BaseService) Add(name string, handler grpctransport.Handler) error {
	_, ok := handlers[name]
	if ok {
		return errors.New("exist already")
	}
	handlers[name] = handler
	return nil
}

func (base *BaseService) AddByType(x proto.Message, handler grpctransport.Handler) error {
	name := proto.MessageName(x)
	return base.Add(name, handler)
}

func (base *BaseService) Execute(name string, ctx context.Context, in interface{}) (rsp interface{}, err error) {
	handler, ok := handlers[name]
	if !ok {
		rsp = nil
		err = errors.New("no matching handler")
		return
	}
	_, rsp, err = handler.ServeGRPC(ctx, in)
	return
}
