package main

import (
	"fmt"
	"local/sndaRpc/cache"
	"local/sndaRpc/client"
	"local/sndaRpc/dbutil"
	"local/sndaRpc/gateway"
	"local/sndaRpc/logHelper"
	"local/sndaRpc/server"
	_ "local/sndaRpc/service"
	"local/sndaRpc/util"
	"os"

	"github.com/astaxie/beego"
	"github.com/go-kit/kit/log/level"
	_ "net/http/pprof"
	"net/http"
	"sync"
)

var (
	xmlconf *util.AppXMLConf
)

func main() {
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
	// level.Error(logger).Log("error", startServer())
	var wg sync.WaitGroup
	wg.Add(3)
	go func(wg *sync.WaitGroup){
		startServer()
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup){
		startHTTPGateway()
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup){
		startDefaultHttpServer()
		wg.Done()
	}(&wg)

	wg.Wait()

}

//加载 app.conf等信息
func initCommonParam() error {
	confPath := beego.AppConfig.DefaultString("xmlconf", "conf/config.xml")
	var err error
	xmlconf, err = util.LoadXMLConfig(confPath)
	return err
}

//初始化log
func initLog() error {
	_, err := logHelper.RegisterByWriter(
		logHelper.ALL,
		os.Stdout,
		beego.AppConfig.DefaultString("log.level.all", "debug"),
	)
	if err != nil {
		return err
	}
	logHelper.Register(
		logHelper.REQUEST_IN,
		beego.AppConfig.DefaultString("log.path.request_in", "logs/request-in.log"),
		beego.AppConfig.DefaultString("log.level.request_in", "debug"),
	)
	if err != nil {
		return err
	}
	logHelper.Register(
		logHelper.REQUEST_OUT,
		beego.AppConfig.DefaultString("log.path.request_out", "logs/request-out.log"),
		beego.AppConfig.DefaultString("log.level.request_out", "debug"),
	)
	if err != nil {
		return err
	}
	logHelper.Register(
		logHelper.DB,
		beego.AppConfig.DefaultString("log.path.db", "logs/db-out.log"),
		beego.AppConfig.DefaultString("log.level.db", "debug"),
	)
	if err != nil {
		return err
	}
	logHelper.Register(
		logHelper.GATE_WAY,
		beego.AppConfig.DefaultString("log.path.http_gateway", "logs/http-gateway.log"),
		beego.AppConfig.DefaultString("log.level.http_gateway", "debug"),
	)
	if err != nil {
		return err
	}
	return nil
}

//连接redis
func initRedis() error {
	allLogger := logHelper.Logger(logHelper.ALL)
	redisManager := cache.DefaultRedisManager()
	redisManager.SetLogger(allLogger)
	redisList := xmlconf.RedisList
	for _, info := range redisList {
		if len(info.Addr[0]) == 0 {
			continue
		}
		if err := redisManager.Register(info.Name, info.Addr[0], info.Passwd, info.PoolSize); err != nil {
			return err
		}
	}
	return nil
}

//连接mysql
func initMysql() error {
	allLogger := logHelper.Logger(logHelper.DB)
	mysqlManager := dbutil.DefaultMySQLManager()
	mysqlManager.SetLogger(allLogger)
	mySqlList := xmlconf.MySQLList
	for _, info := range mySqlList {
		if len(info.Addr[0]) == 0 {
			continue
		}
		if err := mysqlManager.Register(info.Name, info.Addr[0], info.MaxIdleConn, info.MaxOpenConns); err != nil {
			return err
		}
	}
	return nil
}

//创建连接远程服务的客户端
func initClient() error {
	outLogger := logHelper.Logger(logHelper.REQUEST_OUT)
	// allLogger := logHelper.Logger(logHelper.ALL)
	grpcClient := client.DefaultGRPCClient()
	if err := grpcClient.SetLogger(outLogger); nil != err {
		return err
	}
	for _, clientInfo := range xmlconf.ClientList {
		err := grpcClient.Register(clientInfo)
		if err != nil {
			// level.Error(allLogger).Log("error", "can not init client", "reason", err)
			return err
		}
	}
	return nil
}

//启动grpc服务
func startServer() error {
	addr := beego.AppConfig.DefaultString("rpcaddr", ":8081")
	var grpcServer server.Server = server.DefaultGRPCServer()
	grpcServer.SetLogger(logHelper.Logger(logHelper.REQUEST_IN))
	logger := logHelper.Logger(logHelper.ALL)
	//for _, srvInfo := range xmlconf.ServiceList {
	//	err := grpcServer.RegisterByConfig(srvInfo)
	//	if err != nil {
	//		level.Error(logger).Log("during", "register service", "err", err)
	//		os.Exit(1)
	//	}
	//}
	level.Warn(logger).Log("msg", "gRPC server start success", "addr", addr)
	level.Error(logger).Log("error", grpcServer.Serve(addr))
	return nil
}

func startHTTPGateway() error {
	gatewayLogger := logHelper.Logger(logHelper.GATE_WAY)
	allLogger := logHelper.Logger(logHelper.ALL)
	gw := gateway.DefaultHTTPGateWay()
	if err := gw.SetLogger(gatewayLogger); nil != err {
		return err
	}
	err := gw.Register(xmlconf.HTTPGateWayList)
	if err != nil {
		level.Error(allLogger).Log("error", "did not init http gateway", "reason", err)
		return err
	}
	addr := beego.AppConfig.DefaultString("gatewayaddr", ":80")
	level.Warn(allLogger).Log("msg", "http gateway server start success", "addr", addr)
	level.Error(allLogger).Log("error", gw.Serve(addr))
	return nil
}


//启动grpc服务
func startDefaultHttpServer() error {
	addr := beego.AppConfig.DefaultString("httpaddr", ":8083")
	logger := logHelper.Logger(logHelper.ALL)
	level.Warn(logger).Log("msg", "default http server start success", "addr", addr)
	level.Error(logger).Log("error", http.ListenAndServe(addr, nil))
	return nil
}
