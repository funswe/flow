package flow

import (
	"context"
	"errors"
	"fmt"
	"github.com/funswe/flow/utils/json"
	"github.com/go-redis/redis/v8"
	"reflect"
	"time"
)

var ctx = context.Background()

// 定义redis配置结构
type RedisConfig struct {
	Enable   bool
	Password string
	DbNum    int
	Host     string
	Port     int
	Prefix   string
}

// 返回默认的redis配置
func defRedisConfig() *RedisConfig {
	return &RedisConfig{
		Enable: false,
		Host:   "127.0.0.1",
		Port:   6379,
		DbNum:  0,
		Prefix: "flow",
	}
}

const Nil = NotExistError("flow-redis: key not exist")

// 定义redis键不存在错误对象
type NotExistError string

func (e NotExistError) Error() string { return string(e) }

func (NotExistError) NotExistError() {}

// 定义返回的结果
type RedisResult string

// 将返回的结果定义到给定的对象里
func (rr RedisResult) Parse(v interface{}) error {
	return json.Unmarshal([]byte(string(rr)), v)
}

// 返回原始的请求字符串结果数据
func (rr RedisResult) Raw() string {
	return string(rr)
}

// 定义redis操作对象
type RedisClient struct {
	app      *Application
	NotExist NotExistError
	rdb      *redis.Client
}

func (rd *RedisClient) fillKey(key string) string {
	return fmt.Sprintf("%s-%s", rd.app.redisConfig.Prefix, key)
}

func (rd *RedisClient) Close() error {
	if rd.rdb != nil {
		return rd.rdb.Close()
	}
	return nil
}

func (rd *RedisClient) Get(key string) (RedisResult, error) {
	val, err := rd.rdb.Get(ctx, rd.fillKey(key)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", Nil
		}
		return "", err
	}
	return RedisResult(val), nil
}

func (rd *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	switch reflect.TypeOf(value).Kind() {
	case reflect.Ptr, reflect.Map:
		val, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return rd.rdb.Set(ctx, rd.fillKey(key), string(val), expiration).Err()
	case reflect.String:
		return rd.rdb.Set(ctx, rd.fillKey(key), value, expiration).Err()
	default:
		return errors.New("value is neither map nor struct or string")
	}
}

func (rd *RedisClient) Delete(key string) error {
	return rd.rdb.Del(ctx, rd.fillKey(key)).Err()
}

// 返回默认的redis操作对象
func defRedis() *RedisClient {
	return &RedisClient{
		NotExist: Nil,
	}
}

// 初始化redis
func initRedis(app *Application) {
	if app.redisConfig != nil && app.redisConfig.Enable {
		rdb := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", app.redisConfig.Host, app.redisConfig.Port),
			Password: app.redisConfig.Password,
			DB:       app.redisConfig.DbNum,
		})
		app.redis.rdb = rdb
		err := rdb.Ping(ctx).Err()
		if err != nil {
			panic(err)
		}
		app.redis.app = app
		logFactory.Info("redis server init ok")
	}
}
