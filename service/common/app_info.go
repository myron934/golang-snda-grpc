package common

import (
	"local/sndaRpc/inject"
	"local/sndaRpc/pb/common"
	"local/sndaRpc/server"
	"golang.org/x/net/context"
)

func init() {
	inject.Inject((*common.CommonServiceServer)(nil))
	inject.Inject(CommonService{})
	server.DefaultGRPCServer().Register((*common.CommonServiceServer)(nil), CommonService{},
		"/common.commonService",
		"common.proto",
		server.MakeMethodInfoByName("appInfo", "common.appInfoRequest", "common.appInfoReply"),
	)
}

type CommonService struct {
}

func (s *CommonService) AppInfo(ctx context.Context, in *common.AppInfoRequest) (*common.AppInfoReply, error) {
	rsp := new(common.AppInfoReply)
	rsp.AppName = "叨鱼rpc"
	rsp.Version = "1.0.0"
	return rsp, nil
}
