package flow

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/funswe/flow/log"
	"github.com/funswe/flow/utils/files"
)

// 定义服务配置
type ServerConfig struct {
	AppName    string // 应用名称
	Proxy      bool   // 是否是代理模式
	Host       string // 服务启动地址
	Port       int    // 服务端口
	ViewPath   string // 服务端渲染视图文件路径，使用pongo2模板
	StaticPath string // 服务器静态资源路径
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

// 定义日志配置
type LoggerConfig struct {
	LoggerLevel string
	LoggerPath  string
}

// 返回默认的日志配置
func defLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		LoggerLevel: defLoggerLevel(),
		LoggerPath:  defLoggerPath(),
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

// 定义服务的APP
type Application struct {
	reqId        int64         // 请求ID，每次递增1，服务重启就从1开始计数
	rc           chan int64    // 请求ID的传递channel
	logger       *log.Logger   // 日志对象
	serverConfig *ServerConfig // 服务配置
	loggerConfig *LoggerConfig // 日志配置
	ormConfig    *OrmConfig    // 数据库配置
	redisConfig  *RedisConfig  // redis配置
	corsConfig   *CorsConfig   // 跨域配置
	curlConfig   *CurlConfig   // httpclient配置
	middleware   []Middleware  //中间件集合
	orm          *Orm          // 数据库ORM对象，用于数据库操作
	redis        *RedisClient  // redis对象，用户redis操作
	curl         *Curl         // httpclient对象，用于发送http请求，如get，post
}

// 启动服务
func (app *Application) run() error {
	logFactory = log.New(app.loggerConfig.LoggerPath, app.serverConfig.AppName+".log", app.loggerConfig.LoggerLevel)
	app.logger = logFactory.Create(map[string]interface{}{
		"appName":     app.serverConfig.AppName,
		"proxy":       app.serverConfig.Proxy,
		"host":        app.serverConfig.Host,
		"port":        app.serverConfig.Port,
		"viewPath":    app.serverConfig.ViewPath,
		"loggerPath":  app.loggerConfig.LoggerPath,
		"loggerLevel": app.loggerConfig.LoggerLevel,
		"staticPath":  app.serverConfig.StaticPath,
	})
	app.logger.Info("start params")
	if len(app.serverConfig.StaticPath) != 0 {
		if !files.PathExists(app.serverConfig.StaticPath) {
			os.MkdirAll(app.serverConfig.StaticPath, os.ModePerm)
		}
	}
	// 初始化数据库
	initDB(app)
	// 初始化REDIS
	initRedis(app)
	// 初始化curl
	initCurl(app)
	// 启动一个独立的携程处理请求ID的递增
	go func() {
		for {
			app.reqId++
			app.rc <- app.reqId
		}
	}()
	// 添加默认的中间件
	app.middleware = append([]Middleware{func(ctx *Context, next Next) {
		start := time.Now()
		ctx.Logger.Infof("request incoming, method: %s, uri: %s, host: %s, protocol: %s", ctx.GetMethod(), ctx.GetUri(), ctx.GetHost(), ctx.GetProtocol())
		next()
		cost := time.Since(start)
		ctx.Logger.Infof("request completed, cost: %fms, statusCode: %d", float64(cost.Nanoseconds())/1e6, ctx.GetStatusCode())
	}, func(ctx *Context, next Next) {
		ctx.SetHeader(HttpHeaderXPoweredBy, app.serverConfig.AppName)
		if app.corsConfig.Enable {
			ctx.SetHeader(HttpHeaderCorsOrigin, app.corsConfig.AllowOrigin)
			ctx.SetHeader(HttpHeaderCorsMethods, app.corsConfig.AllowedMethods)
			ctx.SetHeader(HttpHeaderCorsHeaders, app.corsConfig.AllowedHeaders)
		}
		next()
	}}, app.middleware...)
	return http.ListenAndServe(fmt.Sprintf("%s:%d", app.serverConfig.Host, app.serverConfig.Port), router)
}

// 添加中间件
func (app *Application) use(m Middleware) *Application {
	app.middleware = append(app.middleware, m)
	return app
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
