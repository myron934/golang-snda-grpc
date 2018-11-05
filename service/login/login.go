package login

import (
	"local/sndaRpc/cache"
	"local/sndaRpc/dbutil"
	"local/sndaRpc/inject"
	"local/sndaRpc/pb/login"
	"local/sndaRpc/server"
	"local/sndaRpc/util"

	"golang.org/x/net/context"
)

func init() {
	inject.Inject((*login.LoginServiceServer)(nil))
	inject.Inject(TestService{})
	server.DefaultGRPCServer().Register((*login.LoginServiceServer)(nil), TestService{},
		"/login.loginService",
		"loginService.proto",
		server.MakeMethodInfoByName("login", "login.loginRequest", "login.loginReply"),
		server.MakeMethodInfoByName("logout", "login.logoutRequest", "login.logoutRequest"),
	)
}

type TestService struct {
}

func (s *TestService) Login(ctx context.Context, in *login.LoginRequest) (*login.LoginReply, error) {
	rsp := new(login.LoginReply)
	rds, err := cache.DefaultRedisManager().Get("redis1")
	if err != nil {
		rsp.Err = err.Error()
		return nil, err
	}
	v, err := rds.Get(in.GetUserName()).Result()
	if err != nil {
		rsp.Err = err.Error()
	}
	rsp.SessionId = in.GetUserName() + "'s session: " + v
	return rsp, nil
}
func (s *TestService) Logout(ctx context.Context, in *login.LogoutRequest) (*login.LogoutReply, error) {
	rsp := new(login.LogoutReply)
	data, err := dbutil.DefaultMySQLManager().Query(
		"global",
		"SELECT * FROM circle_first_ad_more where circle_id=?",
		2,
	)

	if err != nil {
		rsp.Err = err.Error()
		return rsp, nil
	}
	if len(data) > 0 {
		id := util.String(data[0]["ad_id"], "0")
		rsp.Err = "logout from ad_id=" + id
	}

	return rsp, nil
}
