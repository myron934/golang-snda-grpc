package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"local/sndaRpc/client"
	"local/sndaRpc/logHelper"
	"local/sndaRpc/util"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/golang/protobuf/proto"
)

var (
	defaultHTTPGateWay *HTTPGateWay
)

const (
	// MethodName 每个http请求中必带的一些公共参数
	MethodName = "method"
)

func init() {
	defaultHTTPGateWay = NewHTTPGateway()
}

//HTTPGateWay HTTP网关, 负责接受http请求,并调用处理服务处理,把处理结果转成json返回
type HTTPGateWay struct {
	logger    log.Logger
	isRunning bool
	addr      string
	handlers  map[string]string //<methodName,需要调用的内部处理接口>
	serveMux  *http.ServeMux
}

//New 创建对象
func NewHTTPGateway() *HTTPGateWay {
	gw := new(HTTPGateWay)
	gw.SetLogger(log.NewLogfmtLogger(os.Stderr))
	gw.serveMux = http.NewServeMux()
	gw.handlers = make(map[string]string)
	gw.addr = ":80"
	return gw
}

//DefaultHTTPGateWay 返回默认的HTTPGateWay对象
func DefaultHTTPGateWay() *HTTPGateWay {
	return defaultHTTPGateWay
}

//SetLogger 设置logger
func (me *HTTPGateWay) SetLogger(lg log.Logger) error {
	if nil == lg {
		return fmt.Errorf("nil logger not allow")
	}
	me.logger = lg
	return nil
}

//Register 注册handler
func (me *HTTPGateWay) Register(infoList []*util.HTTPGateWayInfo) error {
	for _, info := range infoList {
		ep := me.makeHTTPEndpoint()
		ep = me.logMeddleWare()(ep)
		handler := kithttp.NewServer(
			ep,
			decodeRequest,
			encodeResponse,
		)
		me.serveMux.Handle(info.Name, handler)
		me.handlers[info.Name] = info.Method
		level.Debug(me.logger).Log(":=", "register http gate way", "name", info.Name, "method", info.Method)
	}
	return nil
}

//Serve 启动服务HTTP网关服务
func (me *HTTPGateWay) Serve(addr string) error {
	if len(addr) > 0 {
		me.addr = addr
	}
	if me.isRunning {
		return errors.New("server is running already")
	}
	me.isRunning = true
	defer func() {
		me.isRunning = false
	}()
	return http.ListenAndServe(me.addr, me.serveMux)

}

func (me *HTTPGateWay) makeHTTPEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		reqMap := request.(map[string]interface{})
		//这个是http的method
		method := reqMap[MethodName].(string)
		//这是映射到真正的rpc method
		rpcMethod, ok := me.handlers[method]
		if !ok {
			return nil, fmt.Errorf("can not find method %s", method)
		}
		clt := client.DefaultGRPCClient()
		info := clt.InterfaceInfo(rpcMethod)
		if nil == info {
			return nil, fmt.Errorf("can not find method %s", rpcMethod)
		}
		tp := proto.MessageType(info.ReqType)
		reqObj := reflect.New(tp.Elem()).Interface()
		b, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, reqObj)
		if err != nil {
			return nil, err
		}
		rsp, err := clt.InvokeTimeout(ctx, rpcMethod, reqObj, time.Second*3)
		if err != nil {
			level.Error(me.logger).Log("error", err)
			return nil, err
		}
		return rsp, nil
	}
}

func (me *HTTPGateWay) logMeddleWare() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			ctx = logHelper.ContextWithNewLogInfo(ctx)
			onceLogger := log.With(me.logger, "ts", log.TimestampFormat(time.Now().Local, "2006-01-02 15:04:05.000.000000"))
			logInfo, _ := logHelper.FromContext(ctx)
			onceLogger = log.With(onceLogger, "flowID", logInfo.FlowID)
			b, err := json.Marshal(request)
			if err == nil {
				reqParams := string(b)
				onceLogger = log.With(onceLogger, "request", reqParams)
			}
			response, err := next(ctx, request)
			//记录响应
			defer func(begin time.Time) {
				b, e := json.Marshal(response)
				if e == nil {
					rspParams := string(b)
					onceLogger = log.With(onceLogger, "response", rspParams)
				}
				level.Info(onceLogger).Log("error", err, "took", time.Since(begin))
			}(time.Now())
			return response, err
		}
	}

}

func decodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	method := r.Method
	switch method {
	case "GET":
		return decodeGETRequest(ctx, r)
	case "POST":
		return decodePOSTRequest(ctx, r)
	default:
		return decodePOSTRequest(ctx, r)
	}
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	header := w.Header()
	header.Add("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func decodeGETRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}
	values := r.Form
	m := toMap(values)
	if _, ok := m[MethodName]; !ok {
		m[MethodName] = r.URL.Path
	}
	return m, nil
}

func decodePOSTRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	// fmt.Println(r.URL)
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}
	values := r.Form
	m := toMap(values)
	if _, ok := m[MethodName]; !ok {
		m[MethodName] = r.URL.Path
	}
	// 尝试解析json
	if r.Body == nil {
		return m, nil
	}

	ct := r.Header.Get("Content-Type")
	if ct != "application/json" && ct != "text/json" {
		return m, nil
	}
	bodyReader := r.Body
	if bodyReader == nil {
		return m, nil
	}
	b, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return m, nil
	}
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal(b, &jsonMap)
	appendMap(jsonMap, m)
	return m, nil

}
func appendMap(from, to map[string]interface{}) {
	for k, v := range from {
		to[k] = v
	}
}
func toMap(urlValues url.Values) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range urlValues {
		if 1 == len(v) {
			m[k] = v[0]
		} else {
			m[k] = v
		}
	}
	return m
}
