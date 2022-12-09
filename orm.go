package flow

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"strings"
	"sync"
	"time"

	"github.com/funswe/flow/log"
	"gorm.io/driver/mysql"
)

// OrmConfig 定义数据库配置
type OrmConfig struct {
	Enable   bool
	UserName string
	Password string
	DbName   string
	Host     string
	Port     int
	Pool     *OrmPool
}

// OrmPool 定义数据库连接池配置
type OrmPool struct {
	MaxIdle         int
	MaxOpen         int
	ConnMaxLifeTime int64
	ConnMaxIdleTime int64
}

type Relation struct {
	Model    Model    // 关联的模型
	As       string   // 主表里字段的field name
	Required bool     // true表示INNER JOIN false是LEFT JOIN
	ON       []string // ON条件
	Fields   []string // 查询的字段
}

type Model interface {
	Relation()
	TableName() string
	Alias() string
}

type QueryBuilder[T Model] struct {
	Model      T
	BD         *gorm.DB
	Conditions []map[string]interface{}
	Fields     []string
	OrderBy    string
	Limit      *clause.Limit
	Relations  []Relation
}

func (q *QueryBuilder[T]) Query() (int64, *[]*T, error) {
	if q.BD == nil {
		panic(errors.New("no db server available"))
	}
	tableName := q.Model.TableName()
	if len(q.Model.Alias()) > 0 {
		tableName = fmt.Sprintf("`%s` `%s`", tableName, q.Model.Alias())
	}
	db := q.BD.Table(tableName)
	where := q.fillConditions(q.Conditions)
	for _, v := range where {
		db.Where(v["key"], v["val"])
	}
	selectFields := make([]string, 0)
	if len(q.Fields) == 0 {
		selectFields = append(selectFields, q.getSelectFields(q.Model)...)
	}
	if len(q.Relations) > 0 {
		for _, v := range q.Relations {
			var build strings.Builder
			if v.Required {
				build.WriteString("INNER JOIN ")
			} else {
				build.WriteString("LEFT JOIN ")
			}
			build.WriteString(fmt.Sprintf("`%s` ", v.Model.TableName()))
			if len(v.As) > 0 {
				build.WriteString(fmt.Sprintf("`%s` ", v.As))
			}
			if len(v.ON) > 0 {
				build.WriteString(fmt.Sprintf("ON %s ", v.ON[0]))
			}
			db.Joins(build.String(), v.ON[1:])
			if len(q.Fields) == 0 && len(v.Fields) == 0 {
				selectFields = append(selectFields, q.getJoinSelectFields(v.Model, v.As)...)
			}
		}
	}
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return 0, nil, err
	}
	var result []*T
	if len(q.Fields) > 0 {
		db.Select(q.Fields)
	} else {
		db.Select(selectFields)
	}
	if len(q.OrderBy) > 0 {
		db.Order(q.OrderBy)
	}
	if q.Limit.Offset > 0 {
		db.Offset(q.Limit.Offset)
	}
	if q.Limit.Limit > 0 {
		db.Limit(q.Limit.Limit)
	}
	if err := db.Find(&result).Error; err != nil {
		return 0, nil, err
	}
	return total, &result, nil
}

func (q *QueryBuilder[T]) fillConditions(conditions []map[string]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	for _, v := range conditions {
		r := make(map[string]interface{}, 0)
		key := v["key"]
		if !strings.Contains(key.(string), ".") {
			key = fmt.Sprintf("%s.%s", q.Model.Alias(), key)
		}
		r["key"] = key
		if _, ok := v["val"]; ok {
			r["val"] = v["val"]
		}
		result = append(result, r)
	}
	return result
}

func (q *QueryBuilder[T]) getSelectFields(model Model) []string {
	result := make([]string, 0)
	schema, _ := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	for _, v := range schema.Fields {
		if _, ok := v.TagSettings["NOSELECT"]; ok {
			continue
		}
		if _, ok := v.TagSettings["-"]; ok {
			continue
		}
		result = append(result, fmt.Sprintf("`%s`.`%s`", model.Alias(), v.DBName))
	}
	return result
}

func (q *QueryBuilder[T]) getJoinSelectFields(model Model, as string) []string {
	result := make([]string, 0)
	schema, _ := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	for _, v := range schema.Fields {
		if _, ok := v.TagSettings["NOSELECT"]; ok {
			continue
		}
		if _, ok := v.TagSettings["-"]; ok {
			continue
		}
		result = append(result, fmt.Sprintf("`%s`.`%s` AS `%s__%s`", model.Alias(), v.DBName, as, v.DBName))
	}
	return result
}

// Orm 定义数据库操作对象
type Orm struct {
	app *Application // app对象
	db  *gorm.DB     // grom db对象
}

// DB 返回grom db对象，用于原生查询使用
func (orm *Orm) DB() *gorm.DB {
	if orm.db == nil {
		panic(errors.New("no db server available"))
	}
	return orm.db
}

// 定义数据库logger对象
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

// 返回默认的数据库操logger对象
func defOrmLogger() *dbLogger {
	return &dbLogger{
		LogLevel: logger.Info,
		logger:   log.New(app.loggerConfig.LoggerPath, app.serverConfig.AppName+"_sql.log", "debug"),
	}
}

// 返回默认的数据库操作对象
func defOrm() *Orm {
	return &Orm{}
}

// 返回默认的数据库配置
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

// 返回默认的数据库连接池配置
func defOrmPool() *OrmPool {
	return &OrmPool{
		MaxIdle:         5,
		MaxOpen:         10,
		ConnMaxLifeTime: 30000,
		ConnMaxIdleTime: 10000,
	}
}

// 初始化数据库
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
		app.Orm.db = db
		logFactory.Info("db server init ok")
	}
}
