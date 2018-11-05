package util

import (
	"fmt"
	"os"
	"testing"
)

func Test1(t *testing.T) {
	fmt.Println(Int("-1", 0))
}

func Test2(t *testing.T) {
	s := make([]interface{}, 2)
	s[0] = "Alice"
	s[1] = "Bob"
	fmt.Println(fmt.Sprint(s))
}
func Test3(t *testing.T) {
	workPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println(workPath)
	conf, _ := LoadXMLConfig("../conf/config.xml")
	fmt.Println(conf.ClientList[0].InterfaceList[0].Name)
}

func Test4(t *testing.T) {

}
