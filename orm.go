package flow

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"

	"github.com/funswe/flow/log"
	"gorm.io/driver/mysql"
)

type OrmConfig struct {
	Enable   bool
	UserName string
	Password string
	DbName   string
	Host     string
	Port     int
	Pool     *OrmPool
}

type OrmPool struct {
	MaxIdle         int
	MaxOpen         int
	ConnMaxLifeTime int64
	ConnMaxIdleTime int64
}

type dbLogger struct {
	LogLevel logger.LogLevel
	logger   *log.Logger
}

func (l *dbLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *dbLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.logger.Debugf(msg, data...)
	}
}

func (l *dbLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.logger.Debugf(msg, data...)
	}
}

func (l *dbLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.logger.Debugf(msg, data...)
	}
}

func (l *dbLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel > 0 {
		elapsed := time.Since(begin)
		sql, rows := fc()
		l.logger.Debugf("[%fms] [affected:%d] %s", float64(elapsed.Nanoseconds())/1e6, rows, sql)
	}
}

func defOrmLogger() *dbLogger {
	return &dbLogger{
		LogLevel: logger.Info,
		logger:   log.New(app.loggerConfig.LoggerPath, app.serverConfig.AppName+"_sql.log", "debug"),
	}
}

func defOrmConfig() *OrmConfig {
	return &OrmConfig{
		Enable:   false,
		UserName: "root",
		Password: "root",
		Host:     "127.0.0.1",
		Port:     3306,
		Pool:     defOrmPool(),
	}
}

func defOrmPool() *OrmPool {
	return &OrmPool{
		MaxIdle:         5,
		MaxOpen:         10,
		ConnMaxLifeTime: 30000,
		ConnMaxIdleTime: 10000,
	}
}

func initDB(app *Application) {
	if app.ormConfig != nil && app.ormConfig.Enable {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&loc=Local", app.ormConfig.UserName, app.ormConfig.Password, app.ormConfig.Host, app.ormConfig.Port, app.ormConfig.DbName)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: defOrmLogger(),
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
		app.db = db
	}
}
