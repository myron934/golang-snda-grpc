package client

import (
	"fmt"
	"local/sndaRpc/logHelper"
	"local/sndaRpc/pb/login"
	"local/sndaRpc/util"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/astaxie/beego"
	"github.com/go-kit/kit/log/level"
)

const (
	ADDRESS = "127.0.0.1:8081,127.0.0.1:8081"
)

var mut sync.Mutex

func Test1(t *testing.T) {
	logger, err := logHelper.RegisterByWriter(
		logHelper.ALL,
		os.Stdout,
		beego.AppConfig.DefaultString("log.level.all", "debug"),
	)
	logHelper.Register(
		logHelper.REQUEST_IN,
		beego.AppConfig.DefaultString("log.path.request_in", "../logs/request-in.log"),
		beego.AppConfig.DefaultString("log.level.request_in", "debug"),
	)
	outLogger, err := logHelper.Register(
		logHelper.REQUEST_OUT,
		beego.AppConfig.DefaultString("log.path.request_out", "../logs/request-out.log"),
		beego.AppConfig.DefaultString("log.level.request_out", "debug"),
	)
	if nil != err {
		fmt.Println(err)
		return
	}
	if err = SetLogger(outLogger); nil != err {
		fmt.Println(err)
		return
	}
	var methods []*util.InterfaceInfo
	methods = append(methods, &util.InterfaceInfo{"/login.loginService/login", "login.loginRequest", "login.LoginReply"})
	methods = append(methods, &util.InterfaceInfo{"/login.loginService/logout", "login.logoutRequest", "login.logoutReply"})
	err = RegisterClient2(ADDRESS, methods)
	if err != nil {
		level.Error(logger).Log("error", "did not connect", "reason", err)
		return
	}
	var wg sync.WaitGroup
	n := 1
	wg.Add(n)
	for i := 0; i < n; i++ {
		go request(&wg)
	}
	wg.Wait()

}

func request(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	ctx := logHelper.NewContext()
	r, err := InvokeTimeout(ctx, "/login.loginService/login", &login.LoginRequest{"tommy", "213"}, time.Second*3)
	if err != nil {
		logHelper.Error("all").Log("error", "could not login", "reason", err)
		return
	}
	loginRsp := r.(*login.LoginReply)
	logHelper.Debug("all").Log("_", loginRsp.GetSessionId())

	//r, err = InvokeTimeout("/pb.login.loginService/logout", ctx, &login.LogoutRequest{"alice"}, time.Second*5)
	//if err != nil {
	//	logHelper.Error("all").Log("error", "could not logout", "reason", err)
	//	return
	//}
	//logoutRsp := r.(*login.LogoutReply)
	//logHelper.Debug("all").Log("_", logoutRsp.GetErr())
}
