package flow

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// ServerConfig 定义服务配置
type ServerConfig struct {
	AppName string // 应用名称
	Proxy   bool   // 是否是代理模式
	Host    string // 服务启动地址
	Port    int    // 服务端口
}

// 返回默认的服务配置
func defServerConfig() *ServerConfig {
	return &ServerConfig{
		AppName: defAppName(),
		Proxy:   defProxy(),
		Host:    defHost(),
		Port:    defPort(),
	}
}

// LoggerConfig 定义日志配置
type LoggerConfig struct {
	LoggerLevel  string
	LoggerPath   string
	LoggerMaxAge int64
	FormatJson   bool
}

type BeforeRun func(app *Application)

// AfterRun 服务退出（Run返回）后执行的收尾方法
type AfterRun func(app *Application)

// 返回默认的日志配置
func defLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		LoggerLevel:  defLoggerLevel(),
		LoggerPath:   defLoggerPath(),
		LoggerMaxAge: defLoggerMaxAge(),
		FormatJson:   defLoggerFormatJson(),
	}
}

func defAppName() string {
	return "flow"
}

func defProxy() bool {
	return false
}

func defHost() string {
	return "127.0.0.1"
}

func defPort() int {
	return 9505
}

func defLoggerPath() string {
	path, _ := filepath.Abs(".")
	return filepath.Join(path, "logs")
}

func defLoggerLevel() string {
	return "debug"
}

func defLoggerMaxAge() int64 {
	return 30
}

func defLoggerFormatJson() bool {
	return true
}

// Application 定义服务的APP
type Application struct {
	reqId        int64         // 请求ID，每次递增1，服务重启就从1开始计数
	Logger       *zap.Logger   // 日志对象
	serverConfig *ServerConfig // 服务配置
	loggerConfig *LoggerConfig // 日志配置
	ormConfig    *OrmConfig    // 数据库配置
	redisConfig  *RedisConfig  // redis配置
	corsConfig   *CorsConfig   // 跨域配置
	curlConfig   *CurlConfig   // httpclient配置
	jwtConfig    *JwtConfig    // JWT配置
	Orm          *Orm          // 数据库ORM对象，用于数据库操作
	Redis        *RedisClient  // redis对象，用户redis操作
	Curl         *Curl         // httpclient对象，用于发送http请求，如get，post
	Jwt          *Jwt          // JWT对象
	beforeRuns   []BeforeRun   // 运行前需要执行的函数列表
	afterRuns    []AfterRun    // 退出后需要执行的函数列表（逆序执行，便于按依赖倒序清理）

	// shutdownTimeout 优雅停机等待在途请求的最长时间，<=0 时使用默认值
	shutdownTimeout time.Duration
}

// 启动服务
func (app *Application) run() error {
	app.Logger = getLogger(app, map[string]interface{}{
		"appName":     app.serverConfig.AppName,
		"proxy":       app.serverConfig.Proxy,
		"host":        app.serverConfig.Host,
		"port":        app.serverConfig.Port,
		"loggerPath":  app.loggerConfig.LoggerPath,
		"loggerLevel": app.loggerConfig.LoggerLevel,
	})
	// 初始化数据库
	initDB(app)
	// 初始化REDIS
	initRedis(app)
	// 初始化curl
	initCurl(app)
	// 初始化jwt
	initJwt(app)
	for _, beforeRun := range app.beforeRuns {
		beforeRun(app)
	}
	app.Logger.Info("server started")
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", app.serverConfig.Host, app.serverConfig.Port),
		Handler: router,
	}
	err := app.serve(srv)
	if errors.Is(err, http.ErrServerClosed) {
		// 收到停机信号并完成优雅停机：先执行退出钩子，再以 0 退出。
		// 停机是预期内的退出，直接退出可避免调用方 log.Fatal(nil) 打出 "<nil>" 且返回非 0 退出码。
		app.Logger.Info("server shutdown gracefully")
		app.runAfterRuns()
		_ = app.Logger.Sync()
		os.Exit(0)
	}
	// 启动失败等非停机退出：同样执行退出钩子，并把错误交回调用方
	app.runAfterRuns()
	return err
}

// defShutdownTimeout 优雅停机默认等待时间。取5秒是为了小于 supervisor 等进程管理器
// 默认10秒的 stopwaitsecs，避免还没收尾完就被强杀。
const defShutdownTimeout = 5 * time.Second

// serve 启动HTTP服务，并在收到SIGINT/SIGTERM后优雅停机
func (app *Application) serve(srv *http.Server) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	quit := make(chan struct{})
	defer close(quit)
	go func() {
		select {
		case s := <-sig:
			app.Logger.Info("shutdown signal received", zap.String("signal", s.String()))
			app.shutdown(srv)
		case <-quit:
		}
	}()
	return srv.ListenAndServe()
}

// shutdown 停止接收新请求并等待在途请求结束；超过 shutdownTimeout 则强制关闭
func (app *Application) shutdown(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), app.shutdownTimeoutOrDefault())
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.Logger.Error("graceful shutdown timeout, force close", zap.Error(err))
		_ = srv.Close()
	}
}

func (app *Application) shutdownTimeoutOrDefault() time.Duration {
	if app.shutdownTimeout > 0 {
		return app.shutdownTimeout
	}
	return defShutdownTimeout
}

// 设置服务配置
func (app *Application) setServerConfig(serverConfig *ServerConfig) *Application {
	app.serverConfig = serverConfig
	return app
}

// 设置日志服务
func (app *Application) setLoggerConfig(loggerConfig *LoggerConfig) *Application {
	app.loggerConfig = loggerConfig
	return app
}

// 设置数据库配置
func (app *Application) setOrmConfig(ormConfig *OrmConfig) *Application {
	app.ormConfig = ormConfig
	return app
}

// 设置redis配置
func (app *Application) setRedisConfig(redisConfig *RedisConfig) *Application {
	app.redisConfig = redisConfig
	return app
}

// 设置跨域服务
func (app *Application) setCorsConfig(corsConfig *CorsConfig) *Application {
	app.corsConfig = corsConfig
	return app
}

// 设置httpclient配置
func (app *Application) setCurlConfig(curlConfig *CurlConfig) *Application {
	app.curlConfig = curlConfig
	return app
}

// 设置JWT配置
func (app *Application) setJwtConfig(jwtConfig *JwtConfig) *Application {
	app.jwtConfig = jwtConfig
	return app
}

// 添加运行前需要执行的方法
func (app *Application) addBefore(b BeforeRun) *Application {
	app.beforeRuns = append(app.beforeRuns, b)
	return app
}

// 添加退出后需要执行的方法
func (app *Application) addAfter(a AfterRun) *Application {
	app.afterRuns = append(app.afterRuns, a)
	return app
}

// 设置优雅停机超时
func (app *Application) setShutdownTimeout(d time.Duration) *Application {
	app.shutdownTimeout = d
	return app
}

// runAfterRuns 逆序执行退出钩子：后注册的通常依赖先注册的资源，逆序便于先释放依赖方
func (app *Application) runAfterRuns() {
	for i := len(app.afterRuns) - 1; i >= 0; i-- {
		app.afterRuns[i](app)
	}
}

// GetServerConfig 获取服务配置
func (app *Application) GetServerConfig() *ServerConfig {
	return app.serverConfig
}

// GetLoggerConfig 获取日志服务
func (app *Application) GetLoggerConfig() *LoggerConfig {
	return app.loggerConfig
}

// GetOrmConfig 获取数据库配置
func (app *Application) GetOrmConfig() *OrmConfig {
	return app.ormConfig
}

// GetRedisConfig 获取redis配置
func (app *Application) GetRedisConfig() *RedisConfig {
	return app.redisConfig
}

// GetCorsConfig 获取跨域服务
func (app *Application) GetCorsConfig() *CorsConfig {
	return app.corsConfig
}

// GetCurlConfig 获取httpclient配置
func (app *Application) GetCurlConfig() *CurlConfig {
	return app.curlConfig
}

// GetJwtConfig 获取JWT配置
func (app *Application) GetJwtConfig() *JwtConfig {
	return app.jwtConfig
}
