package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"local/sndaRpc/constant"
	"local/sndaRpc/logHelper"
	"local/sndaRpc/util"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/go-stack/stack"
	"github.com/golang/protobuf/proto"
	jujuratelimit "github.com/juju/ratelimit"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	defaultGRPCClient *GRPCClient
)

// GRPCClient grpc客户端
type GRPCClient struct {
	logger          log.Logger                     //记录请求日志用的logger
	clientEndpoints map[string]endpoint.Endpoint   //<interface name, Endpoint>
	clientInfo      map[string]*util.InterfaceInfo // <interface name, info>
	qps             int
	maxAttempts     int           //每个请求重试次数(maxTime最多重试maxAttempts次)
	maxTime         time.Duration //总重试时间
}

// NewGRPCClient 创建新的 GRPCClient
func NewGRPCClient() *GRPCClient {
	client := GRPCClient{
		logger:          log.NewLogfmtLogger(os.Stderr),
		clientEndpoints: make(map[string]endpoint.Endpoint),
		clientInfo:      make(map[string]*util.InterfaceInfo),
		qps:             1000,
		maxAttempts:     3,
		maxTime:         3 * time.Second,
	}
	return &client
}

func init() {
	defaultGRPCClient = NewGRPCClient()
}

// DefaultGRPCClient 返回默认的GRPCClient实例
func DefaultGRPCClient() *GRPCClient {
	return defaultGRPCClient
}

// SetLogger 设置log. 用来记录client相关日志
//  lg
func (me *GRPCClient) SetLogger(lg log.Logger) error {
	if nil == lg {
		return fmt.Errorf("nil logger not allow")
	}
	me.logger = lg
	return nil
}

//InterfaceInfo 通过接口名查询接口的相关配置信息
func (me *GRPCClient) InterfaceInfo(name string) *util.InterfaceInfo {
	return me.clientInfo[name]
}

//Register  注册客户端连接远程服务
//@param
//addrList 远程服务地址列表
//interfaceList 接口信息, 调用远程服务将使用 interfaceList.Name作为ID
func (me *GRPCClient) Register(clientInfo *util.ClientInfo) error {
	return me.register(clientInfo)
}

func (me *GRPCClient) register(clientInfo *util.ClientInfo) error {
	var connList []*grpc.ClientConn
	for _, addr := range clientInfo.Addr {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			level.Warn(me.logger).Log("msg", "connect error", "address", addr, "reason", err)
			continue
		}
		connList = append(connList, conn)
	}
	if 0 == len(connList) {
		return errors.New("all of the address is unavalibale")
	}
	for _, interfaceInfo := range clientInfo.InterfaceList {
		if _, ok := me.clientEndpoints[interfaceInfo.Name]; ok {
			return fmt.Errorf("%s exist already", interfaceInfo.Name)
		}
		idx := strings.LastIndex(interfaceInfo.Name, "/")
		if 2 > idx {
			return fmt.Errorf("Invalid method name: %s ", interfaceInfo.Name)
		}
		var (
			serviceName = interfaceInfo.Name[1:idx]
			methodName  = interfaceInfo.Name[idx+1:]
		)
		rspType := proto.MessageType(interfaceInfo.RspType)
		if rspType == nil {
			return fmt.Errorf("invalid responseType %s", interfaceInfo.RspType)
		}
		me.clientInfo[interfaceInfo.Name] = interfaceInfo
		rspType = rspType.Elem()
		out := reflect.New(rspType).Interface()
		var endpoints sd.FixedEndpointer
		options := []grpctransport.ClientOption{
			grpctransport.ClientBefore(setFlowID()),
		}
		for _, conn := range connList {
			ep := grpctransport.NewClient(
				conn,
				serviceName,
				methodName,
				encodeGRPCSumRequest,
				decodeGRPCSumResponse,
				out,
				options...,
			).Endpoint()
			//断路器放在限流器前面,免得断路器检测到限流器误判服务有问题
			ep = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
				Name:    methodName,
				Timeout: 30 * time.Second,
			}))(ep)
			//rate应该是 rate个令牌/ms
			limiter := ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(float64(me.qps), int64(me.qps)))
			ep = limiter(ep)
			endpoints = append(endpoints, ep)
		}
		balancer := lb.NewRoundRobin(endpoints)
		retry := lb.Retry(me.maxAttempts, me.maxTime, balancer)
		me.clientEndpoints[interfaceInfo.Name] = retry
	}
	return nil
}

func encodeGRPCSumRequest(_ context.Context, request interface{}) (interface{}, error) {
	return request, nil
}

func decodeGRPCSumResponse(_ context.Context, response interface{}) (interface{}, error) {
	return response, nil
}

func (me *GRPCClient) invoke(ctx context.Context, method string, request interface{}) (response interface{}, err error) {
	onceLogger := log.With(me.logger, "ts", log.TimestampFormat(time.Now().Local, "2006-01-02 15:04:05.000.000000"))
	onceLogger = log.With(onceLogger, "caller", stack.Caller(2))
	onceLogger = log.With(onceLogger, "method", method)
	if logInfo, ok := logHelper.FromContext(ctx); ok {
		onceLogger = log.With(onceLogger, "flowID", logInfo.FlowID)
	}

	if _, ok := me.clientEndpoints[method]; !ok {
		return nil, fmt.Errorf("no matching method %s was found" + method)
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

	response, err = me.clientEndpoints[method](ctx, request)
	return
}

//Invoke 调用某个接口
// ctx 上下文
// method 方法名(接口名)
// request 入参
//return 出参,error
func (me *GRPCClient) Invoke(ctx context.Context, method string, request interface{}) (response interface{}, err error) {
	return me.invoke(ctx, method, request)
}

//InvokeTimeout 同步超时调用某个接口
// ctx 上下文
// method 方法名(接口名)
// request 入参
//return 出参,error
func (me *GRPCClient) InvokeTimeout(ctx context.Context, method string, request interface{}, duration time.Duration) (response interface{}, err error) {
	var cancelFunc context.CancelFunc
	ctx, cancelFunc = context.WithTimeout(ctx, duration)
	response, err = me.Invoke(ctx, method, request)
	cancelFunc()
	return
}

func setFlowID() grpctransport.ClientRequestFunc {
	return func(ctx context.Context, md *metadata.MD) context.Context {
		if logInfo, ok := logHelper.FromContext(ctx); ok {
			key, val := grpctransport.EncodeKeyValue(constant.FLOW_ID, logInfo.FlowID)
			(*md)[key] = append((*md)[key], val)
		}
		return ctx
	}
}
