package inject

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
)

var (
	exitErr       = errors.New("class exist")
	canNotFindErr = errors.New("can not find class")
)

var injectMap map[string]reflect.Type

func init() {
	injectMap = make(map[string]reflect.Type)
}

func Inject(cls interface{}) error {
	tp := reflect.TypeOf(cls)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	//tp := reflect.TypeOf(
	//	reflect.Indirect(
	//		reflect.ValueOf(cls),
	//	).Interface())
	var buf bytes.Buffer
	buf.WriteString(tp.PkgPath())
	buf.WriteByte('/')
	buf.WriteString(tp.Name())
	clsPath := buf.String()
	if _, ok := injectMap[clsPath]; ok {
		return exitErr
	}
	injectMap[clsPath] = tp
	return nil
}

func New(className string) (interface{}, error) {
	if tp, ok := injectMap[className]; ok {
		return reflect.New(tp).Interface(), nil
	}
	return nil, fmt.Errorf("can not find class %s", className)
}

func Type(className string) (reflect.Type, error) {
	if _, ok := injectMap[className]; !ok {
		return nil, fmt.Errorf("can not find class %s", className)
	}
	return injectMap[className], nil
}

