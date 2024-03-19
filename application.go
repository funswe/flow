package flow

import (
	"fmt"
	"github.com/funswe/flow/log"
	"github.com/funswe/flow/utils/files"
	"net/http"
	"os"
	"path/filepath"
)

// ServerConfig 定义服务配置
type ServerConfig struct {
	AppName    string // 应用名称
	Proxy      bool   // 是否是代理模式
	Host       string // 服务启动地址
	Port       int    // 服务端口
	StaticPath string // 服务器静态资源路径
}

// 返回默认的服务配置
func defServerConfig() *ServerConfig {
	return &ServerConfig{
		AppName:    defAppName(),
		Proxy:      defProxy(),
		Host:       defHost(),
		Port:       defPort(),
		StaticPath: defStaticPath(),
	}
}

// LoggerConfig 定义日志配置
type LoggerConfig struct {
	LoggerLevel  string
	LoggerPath   string
	LoggerMaxAge int64
}

type BeforeRun func(app *Application)

// 返回默认的日志配置
func defLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		LoggerLevel:  defLoggerLevel(),
		LoggerPath:   defLoggerPath(),
		LoggerMaxAge: defLoggerMaxAge(),
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

func defStaticPath() string {
	path, _ := filepath.Abs(".")
	return filepath.Join(path, "statics")
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

// Application 定义服务的APP
type Application struct {
	reqId        int64         // 请求ID，每次递增1，服务重启就从1开始计数
	rc           chan int64    // 请求ID的传递channel
	Logger       *log.Logger   // 日志对象
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
}

// 启动服务
func (app *Application) run() error {
	if len(app.serverConfig.StaticPath) == 0 {
		app.serverConfig.StaticPath = "statics"
	}
	router.ServeFiles("/files/*filepath", http.Dir(app.serverConfig.StaticPath))
	logFactory = log.New(app.loggerConfig.LoggerPath, app.serverConfig.AppName+".log", app.loggerConfig.LoggerLevel,
		app.loggerConfig.LoggerMaxAge)
	app.Logger = logFactory.Create(map[string]interface{}{
		"appName":     app.serverConfig.AppName,
		"proxy":       app.serverConfig.Proxy,
		"host":        app.serverConfig.Host,
		"port":        app.serverConfig.Port,
		"loggerPath":  app.loggerConfig.LoggerPath,
		"loggerLevel": app.loggerConfig.LoggerLevel,
		"staticPath":  app.serverConfig.StaticPath,
	})
	app.Logger.Info("start params")
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
	// 初始化jwt
	initJwt(app)
	// 启动一个独立的携程处理请求ID的递增
	go func() {
		for {
			app.reqId++
			app.rc <- app.reqId
		}
	}()
	for _, beforeRun := range app.beforeRuns {
		beforeRun(app)
	}
	return http.ListenAndServe(fmt.Sprintf("%s:%d", app.serverConfig.Host, app.serverConfig.Port), router)
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
