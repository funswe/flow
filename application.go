package flow

import (
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/funswe/flow/log"
	"github.com/funswe/flow/utils/files"
)

type ServerConfig struct {
	AppName    string
	Proxy      bool
	Host       string
	Port       int
	ViewPath   string
	StaticPath string
}

func defServerConfig() *ServerConfig {
	return &ServerConfig{
		AppName: defAppName(),
		Proxy:   defProxy(),
		Host:    defHost(),
		Port:    defPort(),
	}
}

type LoggerConfig struct {
	LoggerLevel string
	LoggerPath  string
}

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

type Application struct {
	reqId        int64
	rc           chan int64
	logger       *log.Logger
	serverConfig *ServerConfig
	loggerConfig *LoggerConfig
	ormConfig    *OrmConfig
	middleware   []Middleware
	db           *gorm.DB
}

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
	// 启动一个独立的携程处理请求ID的递增
	go func() {
		for {
			app.reqId++
			app.rc <- app.reqId
		}
	}()
	app.middleware = append([]Middleware{func(ctx *Context, next Next) {
		start := time.Now()
		ctx.Logger.Infof("request incoming, method: %s, uri: %s, host: %s, protocol: %s", ctx.GetMethod(), ctx.GetUri(), ctx.GetHost(), ctx.GetProtocol())
		next()
		cost := time.Since(start)
		ctx.Logger.Infof("request completed, cost: %fms, statusCode: %d", float64(cost.Nanoseconds())/1e6, ctx.GetStatusCode())
	}, func(ctx *Context, next Next) {
		ctx.SetHeader(HttpHeaderXPoweredBy, app.serverConfig.AppName)
		next()
	}}, app.middleware...)
	return http.ListenAndServe(fmt.Sprintf("%s:%d", app.serverConfig.Host, app.serverConfig.Port), router)
}

func (app *Application) use(m Middleware) *Application {
	app.middleware = append(app.middleware, m)
	return app
}

func (app *Application) setServerConfig(serverConfig *ServerConfig) *Application {
	app.serverConfig = serverConfig
	return app
}

func (app *Application) setLoggerConfig(loggerConfig *LoggerConfig) *Application {
	app.loggerConfig = loggerConfig
	return app
}

func (app *Application) setOrmConfig(ormConfig *OrmConfig) *Application {
	app.ormConfig = ormConfig
	return app
}
