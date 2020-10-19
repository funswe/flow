package flow

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	orm          *Orm
	serverConfig *ServerConfig
	loggerConfig *LoggerConfig
	ormConfig    *OrmConfig
	middleware   []Middleware
}

func (app *Application) initDB() {
	if app.ormConfig != nil && app.ormConfig.Enable {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&loc=Local", app.ormConfig.UserName, app.ormConfig.Password, app.ormConfig.Host, app.ormConfig.Port, app.ormConfig.DbName)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.New(
				logFactory,
				logger.Config{
					SlowThreshold: time.Second, // 慢 SQL 阈值
					LogLevel:      logger.Info, // Log level
					Colorful:      false,       // 禁用彩色打印
				}),
		})
		if err != nil {
			panic(err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			panic(err)
		}
		sqlDB.SetConnMaxIdleTime(time.Duration(app.ormConfig.Pool.ConnMaxIdleTime) * time.Second)
		sqlDB.SetConnMaxLifetime(time.Duration(app.ormConfig.Pool.ConnMaxLifeTime) * time.Second)
		sqlDB.SetMaxIdleConns(app.ormConfig.Pool.MaxIdle)
		sqlDB.SetMaxOpenConns(app.ormConfig.Pool.MaxOpen)
		err = sqlDB.Ping()
		if err != nil {
			panic(err)
		}
		app.orm.db = db
	}
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
	app.initDB()
	// 启动一个独立的携程处理请求ID的递增
	go func() {
		for {
			app.reqId++
			app.rc <- app.reqId
		}
	}()
	app.middleware = append([]Middleware{func(ctx *Context, next Next) {
		start := time.Now().UnixNano()
		ctx.Logger.Debugf("request incoming, method: %s, uri: %s, host: %s, protocol: %s", ctx.GetMethod(), ctx.GetUri(), ctx.GetHost(), ctx.GetProtocol())
		next()
		cost := time.Now().UnixNano() - start
		ctx.Logger.Debugf("request completed, cost: %d ms, statusCode: %d", cost/1000000, ctx.GetStatusCode())
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
