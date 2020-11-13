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

type RedisConfig struct {
	Enable   bool
	Password string
	DbNum    int
	Host     string
	Port     int
}

func defRedisConfig() *RedisConfig {
	return &RedisConfig{
		Enable: false,
		Host:   "127.0.0.1",
		Port:   6379,
		DbNum:  0,
	}
}

const Nil = NotExistError("flow-redis: key not exist")

type NotExistError string

func (e NotExistError) Error() string { return string(e) }

func (NotExistError) NotExistError() {}

type RedisClient struct {
	app      *Application
	NotExist NotExistError
	rdb      *redis.Client
}

func (rd *RedisClient) GetRaw(key string) (string, error) {
	return rd.rdb.Get(ctx, key).Result()
}

func (rd *RedisClient) Get(key string, v interface{}) error {
	val, err := rd.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return Nil
		}
		return err
	}
	return json.Unmarshal([]byte(val), v)
}

func (rd *RedisClient) SetRaw(key string, value string, expiration time.Duration) error {
	return rd.rdb.Set(ctx, key, value, expiration).Err()
}

func (rd *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	switch reflect.TypeOf(value).Kind() {
	case reflect.Ptr, reflect.Map:
		val, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return rd.rdb.Set(ctx, key, string(val), expiration).Err()
	default:
		return errors.New("value is neither map nor struct")
	}
}

func defRedis() *RedisClient {
	return &RedisClient{
		NotExist: Nil,
	}
}

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
		logFactory.Info("redis server init ok")
	}
}
