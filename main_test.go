package main

import (
	"fmt"
	"local/sndaRpc/client"
	"local/sndaRpc/logHelper"
	"local/sndaRpc/pb/login"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/kit/log/level"
	"golang.org/x/net/context"
)

const (
	ADDRESS = "127.0.0.1:8081,127.0.0.1:8081"
)

var mut sync.Mutex

func TestClient(t *testing.T) {
	err := initCommonParam()
	if err != nil {
		panic(fmt.Sprintf("load common parameters error: %s", err))
	}
	if err = initLog(); err != nil {
		panic(fmt.Sprintf("init log error: %s", err))
	}
	logger := logHelper.Logger(logHelper.ALL)
	if err = initMysql(); err != nil {
		level.Error(logger).Log("error", fmt.Sprintf("init mysql error:%s", err))
	}
	if err = initRedis(); err != nil {
		level.Error(logger).Log("error", fmt.Sprintf("init Redis error:%s", err))
	}
	if err = initClient(); err != nil {
		level.Error(logger).Log("error", fmt.Sprintf("init client error:%s", err))
	}
	//=================================================
	for i := 0; i < 1; i = i + 1 {
		doLogin()
		doLogout()
	}

}

func doLogin() {
	ctx := logHelper.NewContext()
	rsp, err := client.DefaultGRPCClient().InvokeTimeout(ctx, "/login.loginService/login", &login.LoginRequest{"tommy", "213"}, time.Second*3)
	if err != nil {
		logHelper.Error("all").Log("error", "could not login", "reason", err)
		return
	}
	loginRsp := rsp.(*login.LoginReply)
	logHelper.Debug("all").Log("_", loginRsp.GetSessionId())

}
func doLogout() {
	rsp, err := client.DefaultGRPCClient().InvokeTimeout(logHelper.NewContext(), "/login.loginService/logout", &login.LogoutRequest{"alice"}, time.Second*5)
	if err != nil {
		logHelper.Error("all").Log("error", "could not logout", "reason", err)
		return
	}
	logoutRsp := rsp.(*login.LogoutReply)
	logHelper.Debug("all").Log("_", logoutRsp.GetErr())

}

type TestServiceServer interface {
	Login(context.Context, *login.LoginRequest) (*login.LoginReply, error)
}
