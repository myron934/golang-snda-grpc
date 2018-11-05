package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"local/sndaRpc/constant"
	"local/sndaRpc/inject"
	"local/sndaRpc/logHelper"
	"local/sndaRpc/util"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	kittransport "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/proto"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"

	"os"

	"github.com/go-kit/kit/log/level"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/transport"
)

var (
	defaultGRPCServer *GRPCServer
)

type methodHandler func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error)

//MethodInfo 包含了方法信息
type MethodInfo struct {
	//方法名. 如 "login"
	Name string
	//入参类型名称, 对应proto生成的go文件中的类型. 如 login.LoginRequest
	ReqType reflect.Type
	//出参类型名称, 对应proto生成的go文件中的类型. 如 login.LoginReply
	RspType reflect.Type
}

//MakeMethodInfo 创建一个methodInfo
func MakeMethodInfo(name string, req, rsp interface{}) *MethodInfo {
	reqType := reflect.TypeOf(req)
	if reqType.Kind() == reflect.Ptr {
		reqType = reqType.Elem()
	}
	rspType := reflect.TypeOf(rsp)
	if rspType.Kind() == reflect.Ptr {
		rspType = rspType.Elem()
	}
	return &MethodInfo{name, reqType, rspType}
}

func MakeMethodInfoByName(name, req, rsp string) *MethodInfo {
	reqType := proto.MessageType(req)
	if reqType == nil {
		panic(fmt.Sprintf("unknown type %s", req))
	}
	if reqType.Kind() == reflect.Ptr {
		reqType = reqType.Elem()
	}

	rspType := proto.MessageType(rsp)
	if rspType == nil {
		panic(fmt.Sprintf("unknown type %s", rsp))
	}
	if rspType.Kind() == reflect.Ptr {
		rspType = rspType.Elem()
	}
	return &MethodInfo{name, reqType, rspType}
}

// GRPCServer gprcServer提供基于grpc协议的服务
type GRPCServer struct {
	logger     log.Logger //记录请求日志用的logger
	baseServer *grpc.Server
}

//NewGRPCServer 创建GRPC服务
func NewGRPCServer() *GRPCServer {
	srv := new(GRPCServer)
	srv.SetLogger(log.NewLogfmtLogger(os.Stderr))
	srv.baseServer = grpc.NewServer()
	return srv
}

//DefaultGRPCServer 创建GRPC服务
func DefaultGRPCServer() *GRPCServer {
	return defaultGRPCServer
}

func init() {
	defaultGRPCServer = NewGRPCServer()
}

//SetLogger 记录请求日志用的logger
func (me *GRPCServer) SetLogger(lg log.Logger) error {
	if nil == lg {
		return fmt.Errorf("nil logger not allow")
	}
	me.logger = lg
	return nil
}

//Serve 启动监听
func (me *GRPCServer) Serve(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return me.baseServer.Serve(listener)
}

//Register 这个是通过xml配置注册服务
//@param
//serverInfo.HandlerInterface: 业务类必须实现一个接口. 如 local/sndaRpc/pb/login/LoginServiceServer
//serverInfo.HandlerCls: 实现了 HandlerInterface 接口的具体业务类. 如 local/sndaRpc/service/login/TestService
//serverInfo.ProtoName: 对应的*.proto定义文件. 如loginService.proto
//serverInfo.Name: 要注册的服务名. 如 /login.loginService
//serverInfo.MethodList: serviceName下的所有methods信息. 如 MethodInfo{"login", "login.loginRequest", "login.loginReply"}
// serverInfo.Name/MethodInfo.Name共同构成grpc真正的接口名
//@return: error
func (me *GRPCServer) RegisterByConfig(serverInfo *util.ServerInfo) error {
	handler, err := inject.New(serverInfo.HandlerCls)
	if err != nil {
		return err
	}
	handlerType, err := inject.New(serverInfo.HandlerInterface)
	if err != nil {
		return err
	}
	methodDescList := make([]grpc.MethodDesc, 0)
	for _, method := range serverInfo.MethodList {
		handler, err := me.makeMethodHandler(serverInfo.Name, method.Name, method.ReqType, method.RspType)
		if err != nil {
			return err
		}
		methodDesc := grpc.MethodDesc{
			MethodName: method.Name,
			Handler:    handler,
		}
		methodDescList = append(methodDescList, methodDesc)
	}
	var serviceDesc = grpc.ServiceDesc{
		ServiceName: serverInfo.Name[1:],
		HandlerType: handlerType,
		Methods:     methodDescList,
		Streams:     []grpc.StreamDesc{},
		Metadata:    serverInfo.ProtoName,
	}
	me.baseServer.RegisterService(&serviceDesc, handler)
	return nil
}

//Register 注册服务
//@param
//handlerInterface  业务类必须实现的一个接口. 如 local/sndaRpc/pb/login/LoginServiceServer
//handlerCls 实现了 handlerInterface 接口的具体业务类. 如 local/sndaRpc/service/login/TestService
//serviceName 要注册的服务名. 如 /login.loginService
//protoName 对应的*.proto定义文件. 如loginService.proto
//methodList serviceName下的所有methods信息. 如 MethodInfo{"login", "login.loginRequest", "login.loginReply"}
//serviceName/MethodInfo.Name共同构成grpc真正的接口名
//@return error
func (me *GRPCServer) Register(handlerInterface, handlerCls interface{}, serviceName, protoName string, methodList ...*MethodInfo) error {
	tp := reflect.TypeOf(handlerCls)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	//得到业务处理类
	handler := reflect.New(tp).Interface()
	methodDescList := make([]grpc.MethodDesc, 0)
	for _, method := range methodList {
		var buf bytes.Buffer
		buf.WriteString(strings.ToUpper(method.Name[:1]))
		buf.WriteString(method.Name[1:])
		methodName := buf.String()
		//注册的时候方法首字母变成大写了, 所以要转一下
		methodHandler, err := me.makeMethodHandler2(serviceName, methodName, method.ReqType, method.RspType)
		if err != nil {
			return err
		}
		methodDesc := grpc.MethodDesc{
			MethodName: method.Name,
			Handler:    methodHandler,
		}
		methodDescList = append(methodDescList, methodDesc)
	}
	var serviceDesc = grpc.ServiceDesc{
		ServiceName: serviceName[1:],
		HandlerType: handlerInterface,
		Methods:     methodDescList,
		Streams:     []grpc.StreamDesc{},
		Metadata:    protoName,
	}
	me.baseServer.RegisterService(&serviceDesc, handler)
	return nil
}

//NewServerEndpoint 创建Endpoint
//executor: 处理业务的类实例
//methodName: 处理业务的函数名称,用于控制反转找到对应函数. func f(ctx context.Context, in interface{}) (interface{}, error)
//return: endpoint.Endpoint, error
func (me *GRPCServer) newServerEndpoint(executor interface{}, methodName string) (endpoint.Endpoint, error) {
	ep, err := newServerEndpoint(executor, methodName)
	if err != nil {
		return nil, err
	}
	return me.logParams(ep)
}

//newDefaultHandler 创建GRPC接口处理器,采用默认的调用配置
//executor: 业务处理类
//methodName: executor中对应的处理函数名 func f(ctx context.Context, in interface{}) (interface{}, error)
//return: 处理器和错误信息
func (me *GRPCServer) newDefaultHandler(executor interface{}, methodName string) (kittransport.Handler, error) {
	ep, err := me.newServerEndpoint(executor, methodName)
	if err != nil {
		return nil, err
	}
	//ep = addendpoint.InstrumentingMiddleware("example", "daoyu", "request_duration_seconds", "Request duration in seconds.")(ep)
	options := []kittransport.ServerOption{
		kittransport.ServerErrorLogger(me.logger),
		kittransport.ServerBefore(getFlowID()),
	}
	var handler kittransport.Handler = kittransport.NewServer(
		ep,
		decodeGRPCRequest,
		encodeGRPCResponse,
		options...,
	)
	return handler, nil
}

//创建Endpoint
//executor: 处理业务的类实例
//methodName: 处理业务的函数名称. func f(ctx context.Context, in interface{}) (interface{}, error)
//return: endpoint.Endpoint, error
func newServerEndpoint(executor interface{}, methodName string) (endpoint.Endpoint, error) {
	rv := reflect.ValueOf(executor)
	_, ok := rv.Type().MethodByName(methodName)
	if !ok {
		return nil, fmt.Errorf("no matching method named %s was found", methodName)
	}
	method := rv.MethodByName(methodName)
	var ep endpoint.Endpoint = func(ctx context.Context, request interface{}) (response interface{}, err error) {
		params := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(request)}
		rspData := method.Call(params)
		if !rspData[0].IsNil() {
			response = rspData[0].Interface()
		}
		if !rspData[1].IsNil() {
			err = rspData[1].Interface().(error)
		}
		return
	}
	ep = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(ep)
	return ep, nil

}

//log params and time took
func (me *GRPCServer) logParams(next endpoint.Endpoint) (endpoint.Endpoint, error) {
	var ep endpoint.Endpoint = func(ctx context.Context, request interface{}) (response interface{}, err error) {
		onceLogger := log.With(me.logger, "ts", log.TimestampFormat(time.Now().Local, "2006-01-02 15:04:05.000.000000"))
		if pr, ok := peer.FromContext(ctx); ok {
			onceLogger = log.With(onceLogger, "addr", pr.Addr.String())
		}
		if logInfo, ok := logHelper.FromContext(ctx); ok {
			onceLogger = log.With(onceLogger, "flowID", logInfo.FlowID)
		}
		if stream, ok := transport.StreamFromContext(ctx); ok {
			onceLogger = log.With(onceLogger, "method", stream.Method())
		}
		b, err := json.Marshal(request)
		if err == nil {
			reqParams := string(b)
			onceLogger = log.With(onceLogger, "request", reqParams)
		}
		//记录响应
		defer func(begin time.Time) {
			b, e := json.Marshal(response)
			if e == nil {
				rspParams := string(b)
				onceLogger = log.With(onceLogger, "response", rspParams)
			}
			level.Info(onceLogger).Log("error", err, "took", time.Since(begin))
		}(time.Now())
		return next(ctx, request)
	}
	return ep, nil

}

//InstrumentingMiddleware 记录请求耗时(metrix)
// InstrumentingMiddleware returns an endpoint middleware that records
// the duration of each invocation to the passed histogram. The middleware adds
// a single field: "success", which is "true" if no error is returned, and
// "false" otherwise.
func InstrumentingMiddleware(namespace string, subsystem string, name string, help string) endpoint.Middleware {
	var duration metrics.Histogram
	{
		// Endpoint-level metrics.
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		}, []string{"success"})
	}
	http.DefaultServeMux.Handle("/metrics", promhttp.Handler())
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func(begin time.Time) {
				duration.With("success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
			}(time.Now())
			return next(ctx, request)
		}
	}
}

func decodeGRPCRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	return grpcReq, nil
}

func encodeGRPCResponse(_ context.Context, response interface{}) (interface{}, error) {
	return response, nil
}

//从context中抽离出客户端传过来的FlowId
func getFlowID() kittransport.ServerRequestFunc {
	return func(ctx context.Context, md metadata.MD) context.Context {
		if flowIDList, ok := md[constant.FLOW_ID]; ok {
			if len(flowIDList) > 0 {
				var logInfo = logHelper.LogInfo{FlowID: flowIDList[0]}
				ctx = logHelper.ContextWithLogInfo(ctx, &logInfo)
			}
		}
		return ctx
	}
}

//makeMethodHandler2 创建方法处理器
//serviceName 服务名 如/login.loginService
//methodName 方法名 如login(首字母小写)
// methodName/methodName 客户端调用时指定的完整接口名. 如 /login.loginService/login(服务名/方法名)
func (me *GRPCServer) makeMethodHandler(serviceName, methodName, reqType, RspType string) (func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error), error) {
	tp := proto.MessageType(reqType)
	if tp == nil {
		return nil, fmt.Errorf("invalid requestType %s", reqType)
	}
	tp = tp.Elem()
	in := reflect.New(tp).Interface()
	handler := func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
		if err := dec(in); err != nil {
			return nil, err
		}
		kitGRPCHandler, err := me.newDefaultHandler(srv, methodName)
		if err != nil {
			return nil, err
		}
		if interceptor == nil {
			_, response, err := kitGRPCHandler.ServeGRPC(ctx, in)
			return response, err
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: serviceName + "/" + methodName,
		}
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			_, response, err := kitGRPCHandler.ServeGRPC(ctx, in)
			return response, err
		}
		return interceptor(ctx, in, info, handler)
	}
	return handler, nil
}

//makeMethodHandler2 创建方法处理器
//serviceName 服务名 如/login.loginService
//methodName 方法名 如Login(首字母大写)
// methodName/methodName 客户端调用时指定的完整接口名. 如 /login.loginService/login(服务名/方法名)
func (me *GRPCServer) makeMethodHandler2(serviceName, methodName string, reqType, rspType reflect.Type) (func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error), error) {
	in := reflect.New(reqType).Interface()
	handler := func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
		if err := dec(in); err != nil {
			return nil, err
		}
		kitGRPCHandler, err := me.newDefaultHandler(srv, methodName)
		if err != nil {
			return nil, err
		}
		if interceptor == nil {
			_, response, err := kitGRPCHandler.ServeGRPC(ctx, in)
			return response, err
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: serviceName + "/" + methodName,
		}
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			_, response, err := kitGRPCHandler.ServeGRPC(ctx, in)
			return response, err
		}
		return interceptor(ctx, in, info, handler)
	}
	return handler, nil
}


//RegisterDefaultServer 注册服务
//@param
//server: *grpc.Server
//handlerInterface: 业务类必须实现一个接口. 如 local/sndaRpc/pb/login/LoginServiceServer
//handlerCls: 实现了interfaceName接口的具体业务类. 如 local/sndaRpc/service/login/TestService
//protoName: 对应的*.proto定义文件. 如loginService.proto
//serviceName: 要注册的服务名. 如 /login.loginService
//methods: serviceName下的所有methods信息. 如 MethodInfo{"login", "login.loginRequest", "login.loginReply"}
//@return: error
//func RegisterDefaultGRPCServer(server *grpc.Server, handlerInterface, handlerCls, protoName, serviceName string, methods []*util.MethodInfo) error {
//	executor, err := inject.New(handlerCls)
//	if err != nil {
//		return err
//	}
//	handlerType, err := inject.New(handlerInterface)
//	if err != nil {
//		return err
//	}
//	methodDescList := make([]grpc.MethodDesc, 0)
//	for _, method := range methods {
//		handler, err := defaultGRPCServer.makeMethodHandler(serviceName, method.Name,method.ReqType,method.RspType)
//		if err != nil {
//			return err
//		}
//		methodDesc := grpc.MethodDesc{
//			MethodName: method.Name,
//			Handler:    handler,
//		}
//		methodDescList = append(methodDescList, methodDesc)
//	}
//	var serviceDesc = grpc.ServiceDesc{
//		ServiceName: serviceName[1:],
//		HandlerType: handlerType,
//		Methods:     methodDescList,
//		Streams:     []grpc.StreamDesc{},
//		Metadata:    protoName,
//	}
//	server.RegisterService(&serviceDesc, executor)
//	return nil
//}
