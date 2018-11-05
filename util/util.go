package util

import (
	"fmt"
	"strconv"
)

func Int(val interface{}, def int) int {
	if v, ok := val.(int); ok {
		return v
	}
	if v, ok := val.(string); ok {
		if v, err := strconv.Atoi(v); err == nil {
			return v
		}
	}
	return def
}

func String(val interface{}, def string) string {
	if nil == val {
		return def
	}
	if v, ok := val.(string); ok {
		return v
	}
	return fmt.Sprint(val)
	//return def
}
