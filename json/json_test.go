package json

import (
	goJson "encoding/json"
	"fmt"
	"testing"

	"github.com/json-iterator/go"
)

func TestJson1(t *testing.T) {
	var val []byte = []byte(`["name","armadillo","age",18]`)
	arr := make([]interface{}, 0)
	fmt.Println(goJson.Unmarshal(val, &arr))
	fmt.Println(arr)

}

func TestJson2(t *testing.T) {
	var val []byte = []byte(`{"name":"armadillo","age":18}`)
	mp := make(map[string]interface{})
	fmt.Println(goJson.Unmarshal(val, &mp))
	fmt.Println(mp)
}

func TestJson3(t *testing.T) {
	val := []byte(`{"ID":1,"Name":"Reds","Colors":["Crimson","Red","Ruby","Maroon"]}`)
	fmt.Println(jsoniter.Get(val, "Colors", 0).ToString())

}

func TestJson4(t *testing.T) {
	m := map[string]interface{}{
		"a": "alice",
		"b": 1,
		"c": 2,
	}
	//json := jsoniter.ConfigCompatibleWithStandardLibrary
	b, err := goJson.Marshal(m)
	fmt.Println(string(b), err)

}
