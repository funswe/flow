package flow

import (
	"context"
	"errors"
	"fmt"
	"github.com/funswe/flow/utils/json"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"reflect"
	"time"
)

var ctx = context.Background()

// RedisConfig 定义redis配置结构
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

func NewNil(key string) error {
	return NotExistError{
		key: key,
	}
}

// NotExistError 定义redis键不存在错误对象
type NotExistError struct {
	key string
}

func (e NotExistError) Error() string {
	return fmt.Sprintf("%s: key not exist", e.key)
}

// RedisResult 定义返回的结果
type RedisResult string

// Parse 将返回的结果定义到给定的对象里
func (rr RedisResult) Parse(v interface{}) error {
	return json.Unmarshal([]byte(string(rr)), v)
}

// Raw 返回原始的请求字符串结果数据
func (rr RedisResult) Raw() string {
	return string(rr)
}

// RedisClient 定义redis操作对象
type RedisClient struct {
	app *Application
	rdb *redis.Client
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
		if errors.Is(err, redis.Nil) {
			return "", NewNil(rd.fillKey(key))
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

func (rd *RedisClient) GetWithOutPrefix(key string) (RedisResult, error) {
	val, err := rd.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", NewNil(rd.fillKey(key))
		}
		return "", err
	}
	return RedisResult(val), nil
}

func (rd *RedisClient) SetWithOutPrefix(key string, value interface{}, expiration time.Duration) error {
	switch reflect.TypeOf(value).Kind() {
	case reflect.Ptr, reflect.Map:
		val, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return rd.rdb.Set(ctx, key, string(val), expiration).Err()
	case reflect.String:
		return rd.rdb.Set(ctx, key, value, expiration).Err()
	default:
		return errors.New("value is neither map nor struct or string")
	}
}

func (rd *RedisClient) DeleteWithOutPrefix(key string) error {
	return rd.rdb.Del(ctx, key).Err()
}

func (rd *RedisClient) IsNil(err error) bool {
	var notExistError NotExistError
	ok := errors.As(err, &notExistError)
	return ok
}

func (rd *RedisClient) GetAllKeys(keyPrefix string) ([]string, error) {
	iter := rd.rdb.Scan(ctx, 0, rd.fillKey(keyPrefix), 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (rd *RedisClient) GetAllKeysWithOutPrefix(keyPrefix string) ([]string, error) {
	iter := rd.rdb.Scan(ctx, 0, keyPrefix, 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (rd *RedisClient) DeleteKeys(keyPrefix string) error {
	keys, err := rd.GetAllKeys(keyPrefix)
	if err != nil {
		return err
	}
	for _, v := range keys {
		err = rd.DeleteKeysWithOutPrefix(v)
	}
	return err
}

func (rd *RedisClient) DeleteKeysWithOutPrefix(keyPrefix string) error {
	keys, err := rd.GetAllKeysWithOutPrefix(keyPrefix)
	if err != nil {
		return err
	}
	for _, v := range keys {
		err = rd.DeleteWithOutPrefix(v)
	}
	return err
}

func (rd *RedisClient) GetClient() *redis.Client {
	return rd.rdb
}

// 初始化redis
func initRedis(app *Application) {
	if app.redisConfig == nil {
		return
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", app.redisConfig.Host, app.redisConfig.Port),
		Password: app.redisConfig.Password,
		DB:       app.redisConfig.DbNum,
	})
	err := rdb.Ping(ctx).Err()
	if err != nil {
		panic(err)
	}
	app.Redis = &RedisClient{
		app: app,
		rdb: rdb,
	}
	app.Logger.Info("redis server started", zap.Any("config", app.redisConfig))
}
