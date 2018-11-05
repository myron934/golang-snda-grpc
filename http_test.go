package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"local/sndaRpc/client"
	"local/sndaRpc/logHelper"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log/level"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/golang/protobuf/proto"
)

func TestHttp(t *testing.T) {
	err := initCommonParam()
	if err != nil {
		panic(fmt.Sprintf("load common parameters error: %s", err))
	}
	if err = initLog(); err != nil {
		panic(fmt.Sprintf("init log error: %s", err))
	}
	logger := logHelper.Logger(logHelper.ALL)
	if err = initClient(); err != nil {
		level.Error(logger).Log("error", fmt.Sprintf("init client error:%s", err))
	}
	handler := kithttp.NewServer(
		makeHTTPEndpoint(),
		decodeRequest,
		encodeResponse,
	)
	http.Handle("/method", handler)
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func makeHTTPEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		reqMap := request.(map[string]interface{})
		method := reqMap["method"].(string)
		info := client.DefaultGRPCClient().InterfaceInfo(method)
		if nil == info {
			return nil, fmt.Errorf("can not find method %s", method)
		}
		tp := proto.MessageType(info.ReqType)
		reqObj := reflect.New(tp)
		b, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, reqObj)
		if err != nil {
			return nil, err
		}
		rsp, err := client.DefaultGRPCClient().InvokeTimeout(logHelper.NewContext(), method, reqObj, time.Second*3)
		if err != nil {
			logHelper.Error("all").Log("error", "could not login", "reason", err)
			return nil, err
		}
		return rsp, nil
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
	if _, ok := m["method"]; !ok {
		m["method"] = r.URL.Path
	}
	return m, nil
}

func decodePOSTRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	fmt.Println(r.URL)
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}
	values := r.Form
	m := toMap(values)
	if _, ok := m["method"]; !ok {
		m["method"] = r.URL.Path
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
