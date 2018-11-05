package json

import (
	"encoding/xml"
	"fmt"
	"os"
	"testing"
)

type AppXMLConf struct {
	XMLName xml.Name       `xml:"config"`
	Serv    []*ServiceInfo `xml:"service"`
	Redis   []*RedisInfo   `xml:"redis"`
	MySql   []*MysqlInfo   `xml:"mysql"`
}

type ServiceInfo struct {
	Name string   `xml:"name,attr"`
	Addr []string `xml:"addr"`
}

type RedisInfo struct {
	Name   string   `xml:"name,attr"`
	Passwd string   `xml:"passwd,attr"`
	Addr   []string `xml:"addr"`
}

type MysqlInfo struct {
	Name         string   `xml:"name,attr"`
	MaxIdleConn  int      `xml:"maxIdleConn,attr"`
	MaxOpenConns int      `xml:"maxOpenConns,attr"`
	Addr         []string `xml:"addr"`
}

func TestXml1(t *testing.T) {
	file, err := os.OpenFile("../conf/config.xml", os.O_RDONLY, os.ModePerm|os.ModeType)
	if err != nil {
		t.Fatal(err)
	}
	dec := xml.NewDecoder(file)
	xc := AppXMLConf{}
	err = dec.Decode(&xc)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("data:%v", xc)
	fmt.Printf("data:%v", xc.Serv)
}
