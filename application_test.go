package flow

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

// 配置设置后能被读取回来（服务/日志/数据库/redis/跨域/httpclient/jwt）
func TestConfigSettersAndGetters(t *testing.T) {
	original := *app
	defer func() { *app = original }()

	SetServerConfig(&ServerConfig{AppName: "unit-test", Host: "127.0.0.1", Port: 8080, Proxy: true})
	SetLoggerConfig(&LoggerConfig{LoggerLevel: "info", LoggerPath: "logs", LoggerMaxAge: 7})
	SetOrmConfig(&OrmConfig{UserName: "root", Password: "pwd", DbName: "db", Host: "127.0.0.1", Port: 3306,
		Pool: &OrmPool{MaxIdle: 5, MaxOpen: 10, ConnMaxLifeTime: 60, ConnMaxIdleTime: 10}})
	SetRedisConfig(&RedisConfig{Enable: true, Host: "127.0.0.1", Port: 6379, DbNum: 1, Prefix: "flow"})
	SetCorsConfig(&CorsConfig{AllowOrigin: "*", AllowedMethods: "GET,POST", AllowedHeaders: "Content-Type"})
	SetCurlConfig(&CurlConfig{Timeout: 3 * time.Second, Headers: map[string]string{"X-Test": "1"}})
	SetJwtConfig(&JwtConfig{Timeout: time.Hour, SecretKey: "secret"})

	if got := app.GetServerConfig(); got == nil || got.AppName != "unit-test" || got.Port != 8080 {
		t.Fatalf("服务配置读取不一致：%+v", got)
	}
	if got := app.GetLoggerConfig(); got == nil || got.LoggerLevel != "info" || got.LoggerMaxAge != 7 {
		t.Fatalf("日志配置读取不一致：%+v", got)
	}
	if got := app.GetOrmConfig(); got == nil || got.DbName != "db" || got.Pool == nil || got.Pool.MaxOpen != 10 {
		t.Fatalf("数据库配置读取不一致：%+v", got)
	}
	if got := app.GetRedisConfig(); got == nil || got.Prefix != "flow" || got.DbNum != 1 {
		t.Fatalf("redis配置读取不一致：%+v", got)
	}
	if got := app.GetCorsConfig(); got == nil || got.AllowOrigin != "*" {
		t.Fatalf("跨域配置读取不一致：%+v", got)
	}
	if got := app.GetCurlConfig(); got == nil || got.Timeout != 3*time.Second || got.Headers["X-Test"] != "1" {
		t.Fatalf("httpclient配置读取不一致：%+v", got)
	}
	if got := app.GetJwtConfig(); got == nil || got.SecretKey != "secret" {
		t.Fatalf("jwt配置读取不一致：%+v", got)
	}
}

// 启动钩子按注册顺序执行、退出钩子逆序执行
func TestRunHooksOrder(t *testing.T) {
	originalBefore, originalAfter := app.beforeRuns, app.afterRuns
	app.beforeRuns, app.afterRuns = nil, nil
	defer func() { app.beforeRuns, app.afterRuns = originalBefore, originalAfter }()

	var order []string
	AddBefore(func(*Application) { order = append(order, "before-1") })
	AddBefore(func(*Application) { order = append(order, "before-2") })
	AddAfter(func(*Application) { order = append(order, "after-1") })
	AddAfter(func(*Application) { order = append(order, "after-2") })

	for _, hook := range app.beforeRuns {
		hook(app)
	}
	app.runAfterRuns()

	expected := []string{"before-1", "before-2", "after-2", "after-1"}
	if len(order) != len(expected) {
		t.Fatalf("钩子执行数量不符：%v", order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf("钩子执行顺序不符：%v，期望 %v", order, expected)
		}
	}
}

// 优雅停机超时：默认5秒，可覆盖
func TestShutdownTimeoutDefault(t *testing.T) {
	original := app.shutdownTimeout
	defer func() { app.shutdownTimeout = original }()

	app.shutdownTimeout = 0
	if got := app.shutdownTimeoutOrDefault(); got != defShutdownTimeout {
		t.Fatalf("默认停机超时应为 %v，实际 %v", defShutdownTimeout, got)
	}
	SetShutdownTimeout(0)
	if got := app.shutdownTimeoutOrDefault(); got != defShutdownTimeout {
		t.Fatalf("传0应回退默认值，实际 %v", got)
	}
}

// 默认的各组件配置都能正常构建
func TestDefaultComponentConfigs(t *testing.T) {
	useTempLogDir(t)
	if cfg := defJwtConfig(); cfg == nil || cfg.Timeout <= 0 {
		t.Fatalf("默认jwt配置不合法：%+v", cfg)
	}
	if cfg := defCurlConfig(); cfg == nil || cfg.Timeout <= 0 {
		t.Fatalf("默认curl配置不合法：%+v", cfg)
	}
	if cfg := defCorsConfig(); cfg == nil || len(cfg.AllowOrigin) == 0 {
		t.Fatalf("默认跨域配置不合法：%+v", cfg)
	}
	if cfg := defRedisConfig(); cfg == nil || cfg.Port <= 0 {
		t.Fatalf("默认redis配置不合法：%+v", cfg)
	}
	if cfg := defOrmConfig(); cfg == nil || cfg.Pool == nil {
		t.Fatalf("默认数据库配置不合法：%+v", cfg)
	}
	if logger := getLogger(app, map[string]interface{}{"k": "v"}); logger == nil {
		t.Fatal("getLogger不应返回nil")
	} else {
		_ = logger.Sync()
	}
	_ = zap.NewNop()
}
