package cache

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-redis/redis"
)

var defaultRedisManager *RedisManager

func init() {
	defaultRedisManager = NewRedisManager()
}

// DefaultRedisManager 返回默认redis实例
func DefaultRedisManager() *RedisManager {
	return defaultRedisManager
}

// NewRedisManager 新redis实例
func NewRedisManager() *RedisManager {
	mng := RedisManager{
		logger:       log.NewLogfmtLogger(os.Stderr),
		clientMapper: make(map[string]*redis.Client),
	}
	return &mng
}

// RedisManager redis管理类
type RedisManager struct {
	clientMapper map[string]*redis.Client
	logger       log.Logger
}

//Register 注册redis
func (me *RedisManager) Register(redisName, addr, passwd string, poolSize int) error {
	if _, ok := me.clientMapper[redisName]; ok {
		return fmt.Errorf("redis named %s has exist", redisName)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: passwd, // no password set
		DB:       0,      // use default DB
		PoolSize: poolSize,
	})
	me.clientMapper[redisName] = client
	return nil
}

// RegisterByClient 通过client注册
func (me *RedisManager) RegisterByClient(redisName string, client *redis.Client) error {
	if _, ok := me.clientMapper[redisName]; ok {
		return fmt.Errorf("redis named %s has exist", redisName)
	}
	return nil
}

//Get 获取内容
func (me *RedisManager) Get(redisName string) (*redis.Client, error) {
	if client, ok := me.clientMapper[redisName]; ok {
		return client, nil
	}
	return nil, fmt.Errorf("can not found client named %s", redisName)
}

//SetLogger 设置logger
func (me *RedisManager) SetLogger(logger log.Logger) error {
	if logger == nil {
		return errors.New("nil logger not allowed")
	}
	me.logger = logger
	//redis.SetLogger()
	return nil
}
