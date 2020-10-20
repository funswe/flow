package flow

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"

	"github.com/funswe/flow/log"
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

type Orm struct {
	db *gorm.DB
}

func (o *Orm) Create(value interface{}) error {
	if o.db == nil {
		panic(errors.New("no db server available"))
	}
	if err := o.db.Create(value).Error; err != nil {
		return err
	}
	return nil
}

func (o *Orm) Find(value interface{}) error {
	if err := o.db.Find(value).Error; err != nil {
		return err
	}
	return nil
}

func defOrm() *Orm {
	return &Orm{}
}

type Model struct {
	ID          uint
	UpdatedTime uint
	CreatedTime uint
}

func (m *Model) AddItem() (*Model, error) {
	result := app.orm.db.Create(m)
	if result.Error != nil {
		return nil, result.Error
	}
	return m, nil
}
