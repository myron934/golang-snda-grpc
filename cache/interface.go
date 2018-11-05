package cache

import "github.com/go-kit/kit/log"

//Manager cahce接口, 但是好像统一不了
type Manager interface {
	Register(redisName, addr, passwd string, poolSize int) error
	SetLogger(logger log.Logger) error
}
