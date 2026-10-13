package flow

import (
	"strings"
	"testing"
)

// 默认redis配置
func TestDefaultRedisConfig(t *testing.T) {
	config := defRedisConfig()
	if config == nil || len(config.Host) == 0 || config.Port <= 0 {
		t.Fatalf("默认redis配置不合法：%+v", config)
	}
}

// 键前缀：统一拼接
func TestRedisFillKey(t *testing.T) {
	original := app.redisConfig
	app.redisConfig = &RedisConfig{Prefix: "weims"}
	defer func() { app.redisConfig = original }()

	client := &RedisClient{app: app}
	if got := client.fillKey("user:1"); got != "weims-user:1" {
		t.Fatalf("键前缀拼接不符：%s", got)
	}
}

// 未初始化连接时关闭是安全的
func TestRedisCloseWithoutClient(t *testing.T) {
	client := &RedisClient{app: app}
	if err := client.Close(); err != nil {
		t.Fatalf("未初始化连接时关闭不应报错：%v", err)
	}
}

// 键不存在错误：可用于业务判断
func TestRedisNotExistError(t *testing.T) {
	err := NewNil("user:1")
	if err == nil {
		t.Fatal("NewNil不应返回nil")
	}
	if !strings.Contains(err.Error(), "user:1") || !strings.Contains(err.Error(), "key not exist") {
		t.Fatalf("错误信息不符：%s", err.Error())
	}
	notExist, ok := err.(NotExistError)
	if !ok || notExist.key != "user:1" {
		t.Fatalf("错误类型不符：%#v", err)
	}
}

// RedisResult：原始字符串与JSON解析
func TestRedisResultParseAndRaw(t *testing.T) {
	result := RedisResult(`{"uid":1001,"name":"flow"}`)
	if result.Raw() != `{"uid":1001,"name":"flow"}` {
		t.Fatalf("Raw不符：%s", result.Raw())
	}

	var target struct {
		Uid  int64  `json:"uid"`
		Name string `json:"name"`
	}
	if err := result.Parse(&target); err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if target.Uid != 1001 || target.Name != "flow" {
		t.Fatalf("解析结果不符：%+v", target)
	}

	if err := RedisResult("not-json").Parse(&target); err == nil {
		t.Fatal("非法JSON应返回错误")
	}
}

// 错误判断：键不存在错误会被识别，nil不会被误判
func TestRedisIsNil(t *testing.T) {
	client := &RedisClient{app: app}
	if client.IsNil(nil) {
		t.Fatal("nil不应被判断为键不存在")
	}
	if !client.IsNil(NewNil("user:1")) {
		t.Fatal("键不存在错误应被识别")
	}
}
