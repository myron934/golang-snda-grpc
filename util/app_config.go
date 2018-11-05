package util

import (
	"encoding/xml"
	"fmt"
	"os"
)

//InterfaceInfo client的配置
type InterfaceInfo struct {
	//方法名. 如 "/login.loginService/login"
	Name string `xml:"name,attr" json:"name,omitempty"`
	//入参类型名称, 对应proto生成的go文件中的类型. 如 login.loginRequest
	ReqType string `xml:"request-type,attr" json:"req_type,omitempty"`
	//出参类型名称, 对应proto生成的go文件中的类型. 如 login.loginReply
	RspType string `xml:"response-type,attr" json:"rsp_type,omitempty"`
}

//MethodInfo <method name="/login.loginService/login" request-type="login.loginRequest" response-type="login.loginReply"/>
type MethodInfo struct {
	//方法名. 如 "/login.loginService/login"
	Name string `xml:"name,attr" json:"name,omitempty"`
	//入参类型名称, 对应proto生成的go文件中的类型. 如 login.loginRequest
	ReqType string `xml:"request-type,attr" json:"req_type,omitempty"`
	//出参类型名称, 对应proto生成的go文件中的类型. 如 login.loginReply
	RspType string `xml:"response-type,attr" json:"rsp_type,omitempty"`
}

//ServiceInfo 服务接口注册信息
//<service name="/login.loginService"
//proto-name="loginService.proto"
//handler-interface="local/sndaRpc/pb/login/LoginServiceServer"
//handler-class="local/sndaRpc/service/login/TestService">
//<method name="login" request-type="login.loginRequest" response-type="login.loginReply"/>
//<method name="logout" request-type="login.logoutRequest" response-type="login.logoutReply"/>
//</service>
type ServerInfo struct {
	Name             string        `xml:"name,attr" json:"name,omitempty"`
	ProtoName        string        `xml:"proto-name,attr" json:"proto_name,omitempty"`
	HandlerInterface string        `xml:"handler-interface,attr" json:"handler_interface,omitempty"`
	HandlerCls       string        `xml:"handler-class,attr" json:"handler_cls,omitempty"`
	MethodList       []*MethodInfo `xml:"method" json:"method_list,omitempty"`
}

// ClientInfo 客户端接口注册信息
//<client name="serv">
//<addr>127.0.0.1:8081</addr>
//<addr>127.0.0.1:8081</addr>
//<addr>127.0.0.1:8081</addr>
//<interface name="/login.loginService/login" request-type="login.loginRequest" response-type="login.loginReply"/>
//<interface name="/login.loginService/logout" request-type="login.logoutRequest" response-type="login.logoutReply"/>
//</client>
type ClientInfo struct {
	Name          string           `xml:"name,attr" json:"name,omitempty"`
	Addr          []string         `xml:"addr" json:"addr,omitempty"`
	InterfaceList []*InterfaceInfo `xml:"interface" json:"interface_list,omitempty"`
}

// RedisInfo redis配置信息
//<redis name="redis1" passwd="" poolsize="0">
//<addr>127.0.0:11511</addr>
//<addr>127.0.0:11511</addr>
//<addr>127.0.0:11511</addr>
//</redis>
type RedisInfo struct {
	Name     string   `xml:"name,attr" json:"name,omitempty"`
	Passwd   string   `xml:"passwd,attr" json:"passwd,omitempty"`
	PoolSize int      `xml:"poolsize,attr" json:"pool_size,omitempty"`
	Addr     []string `xml:"addr" json:"addr,omitempty"`
}

//MySQLInfo mysql信息
//<mysql name="mysql1">
//<addr>userplatform:userplatform@tcp(127.0.0.1:3306)/userplatform_global?charset=utf8</addr>
//<addr>userplatform:userplatform@tcp(127.0.0.1:3306)/userplatform_global?charset=utf8</addr>
//<addr>userplatform:userplatform@tcp(127.0.0.1:3306)/userplatform_global?charset=utf8</addr>
//</mysql>
type MySQLInfo struct {
	Name         string   `xml:"name,attr" json:"name,omitempty"`
	MaxIdleConn  int      `xml:"maxIdleConn,attr" json:"max_idle_conn,omitempty"`
	MaxOpenConns int      `xml:"maxOpenConns,attr" json:"max_open_conns,omitempty"`
	Addr         []string `xml:"addr" json:"addr,omitempty"`
}

// HTTPGateWayInfo http网关注册信息
type HTTPGateWayInfo struct {
	Name   string `xml:"name,attr" json:"name,omitempty"`
	Method string `xml:"method,attr" json:"method,omitempty"`
}

// AppXMLConf xml配置信息
type AppXMLConf struct {
	XMLName         xml.Name           `xml:"config" json:"xml_name,omitempty"`
	PathList        []string           `xml:"include" json:"path_list,omitempty"`
	ServiceList     []*ServerInfo      `xml:"service" json:"service_list,omitempty"`
	ClientList      []*ClientInfo      `xml:"client" json:"client_list,omitempty"`
	RedisList       []*RedisInfo       `xml:"redis" json:"redis_list,omitempty"`
	MySQLList       []*MySQLInfo       `xml:"mysql" json:"my_sql_list,omitempty"`
	HTTPGateWayList []*HTTPGateWayInfo `xml:"http>interface" json:"http_gate_way_list,omitempty"`
}

func (me *AppXMLConf) merge(other *AppXMLConf) {
	if nil == other {
		return
	}
	if len(other.ServiceList) > 0 {
		for _, serviceInfo := range other.ServiceList {
			me.ServiceList = append(me.ServiceList, serviceInfo)
		}
	}
	if len(other.ClientList) > 0 {
		for _, clientList := range other.ClientList {
			me.ClientList = append(me.ClientList, clientList)
		}
	}
	if len(other.RedisList) > 0 {
		for _, redisList := range other.RedisList {
			me.RedisList = append(me.RedisList, redisList)
		}
	}
	if len(other.MySQLList) > 0 {
		for _, mySqlList := range other.MySQLList {
			me.MySQLList = append(me.MySQLList, mySqlList)
		}
	}
	if len(other.HTTPGateWayList) > 0 {
		for _, list := range other.HTTPGateWayList {
			me.HTTPGateWayList = append(me.HTTPGateWayList, list)
		}
	}
}

// LoadXMLConfig 加载xml配置
func LoadXMLConfig(path string) (*AppXMLConf, error) {
	conf, err := loadXML(path)
	if err != nil {
		return nil, err
	}
	for _, otherFile := range conf.PathList {
		otherConf, err := loadXML(otherFile)
		if err != nil {
			return nil, fmt.Errorf("loadXML %s error: %s", otherFile, err.Error())
		}
		conf.merge(otherConf)
	}
	return conf, nil
}

func loadXML(path string) (*AppXMLConf, error) {

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm|os.ModeType)
	if err != nil {
		return nil, err
	}
	defer func() {
		file.Close()
	}()
	dec := xml.NewDecoder(file)
	xc := AppXMLConf{}
	err = dec.Decode(&xc)
	if err != nil {
		return nil, err
	}
	return &xc, nil
}
