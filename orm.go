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

const NoSelect = "NOSELECT"

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
	Model     Model         // 关联的模型
	As        string        // 主表里字段的field name
	Required  bool          // true表示INNER JOIN false是LEFT JOIN
	ON        []string      // ON条件
	Fields    []string      // 查询的字段
	Relations []SubRelation // 关联
	GroupBy   string
	OrderBy   string
}

type SubRelation struct {
	Model    Model    // 关联的模型
	As       string   // 主表里字段的field name
	Required bool     // true表示INNER JOIN false是LEFT JOIN
	ON       []string // ON条件
	Fields   []string // 查询的字段
	GroupBy  string
	OrderBy  string
}

type Model interface {
	TableName() string
	Alias() string
}

type QueryBuilder[T Model] struct {
	Model      T
	DB         *gorm.DB
	Conditions []map[string]interface{}
	Fields     []string
	OrderBy    string
	Limit      clause.Limit
	Relations  []Relation
	GroupBy    string
}

func (q *QueryBuilder[T]) FindOne() (*T, error) {
	if q.DB == nil {
		panic(errors.New("no db server available"))
	}
	tableName := q.Model.TableName()
	if len(q.Model.Alias()) > 0 {
		tableName = fmt.Sprintf("`%s` `%s`", tableName, q.Model.Alias())
	}
	db := q.DB.Table(tableName)
	db = q.fillConditions(db, q.Conditions)
	selectFields := make([]string, 0)
	if len(q.Fields) == 0 {
		selectFields = append(selectFields, q.GetSelectFields(q.Model)...)
	} else {
		selectFields = append(selectFields, q.Fields...)
	}
	db = q.handleRelations(db, q.Model, q.Relations)
	var result T
	if len(q.Fields) > 0 {
		db.Select(db.Statement.Selects, q.Fields)
	} else {
		db.Select(db.Statement.Selects, selectFields)
	}
	if len(q.OrderBy) > 0 {
		db.Order(q.OrderBy)
	} else {
		if len(q.Model.Alias()) > 0 {
			db.Order(fmt.Sprintf("`%s`.`id`", q.Model.Alias()))
		} else {
			db.Order(fmt.Sprintf("`%s`.`id`", q.Model.TableName()))
		}
	}
	if len(q.GroupBy) > 0 {
		db.Group(q.GroupBy)
	}
	if err := db.Take(result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func (q *QueryBuilder[T]) Query() (int64, *[]T, error) {
	if q.DB == nil {
		panic(errors.New("no db server available"))
	}
	tableName := q.Model.TableName()
	if len(q.Model.Alias()) > 0 {
		tableName = fmt.Sprintf("`%s` `%s`", tableName, q.Model.Alias())
	} else {
		tableName = fmt.Sprintf("`%s` `%s`", tableName, q.Model.TableName())
	}
	countDB := q.DB.Table(tableName)
	countDB = q.fillConditions(countDB, q.Conditions)
	countDB = q.handleHasOneRelations(countDB, q.Model, q.Relations)
	var total int64
	err := countDB.Count(&total).Error
	if err != nil {
		return 0, nil, err
	}
	db := q.DB.Table(tableName)
	db = q.fillConditions(db, q.Conditions)
	db = q.handleRelations(db, q.Model, q.Relations)
	var result []T
	if len(q.Fields) > 0 {
		db.Select(db.Statement.Selects, q.Fields)
	} else {
		db.Select(db.Statement.Selects, q.GetSelectFields(q.Model))
	}
	if len(q.OrderBy) > 0 {
		db.Order(q.OrderBy)
	}
	if len(q.GroupBy) > 0 {
		db.Group(q.GroupBy)
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

func (q *QueryBuilder[T]) handleRelations(db *gorm.DB, mainModel Model, relations []Relation) *gorm.DB {
	for _, v := range relations {
		mainSchema, err := schema.Parse(mainModel, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			continue
		}
		relation, ok := mainSchema.Relationships.Relations[v.As]
		if !ok {
			continue
		}
		if relation.Type == schema.HasOne {
			db, isHasOne := q.buildHasOneSql(db, mainModel, v)
			if len(v.Relations) > 0 && isHasOne {
				db = q.handleHasOneSubRelations(db, v.Model, v.Relations)
			}
		} else if relation.Type == schema.HasMany {
			hasManyFunc := func(db *gorm.DB) *gorm.DB {
				if len(v.ON) > 0 {
					if len(v.ON) > 1 {
						db.Where(v.ON[0], v.ON[1:])
					} else {
						db.Where(v.ON[0])
					}
				}
				if len(v.Fields) > 0 {
					db.Select(db.Statement.Selects, v.Fields)
				}
				if len(v.OrderBy) > 0 {
					db.Order(v.OrderBy)
				}
				if len(v.GroupBy) > 0 {
					db.Group(v.GroupBy)
				}
				return db
			}
			if len(v.Relations) > 0 {
				hasManyFunc = q.handleHasManyRelations(v)
			}
			db = db.Preload(v.As, hasManyFunc)
		}
	}
	return db
}

func (q *QueryBuilder[T]) handleHasManyRelations(r Relation) func(db *gorm.DB) *gorm.DB {
	for _, v := range r.Relations {
		mainSchema, err := schema.Parse(r.Model, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			continue
		}
		relation, ok := mainSchema.Relationships.Relations[v.As]
		if !ok {
			continue
		}
		if relation.Type == schema.HasOne {
			var build strings.Builder
			if v.Required {
				build.WriteString("INNER JOIN ")
			} else {
				build.WriteString("LEFT JOIN ")
			}
			build.WriteString(fmt.Sprintf("`%s` ", v.Model.TableName()))
			if len(v.Model.Alias()) > 0 {
				build.WriteString(fmt.Sprintf("`%s` ", v.Model.Alias()))
			} else {
				build.WriteString(fmt.Sprintf("`%s` ", v.Model.TableName()))
			}
			subSchema, err := schema.Parse(v.Model, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				continue
			}
			foreignKey, ok1 := subSchema.FieldsByName[relation.Field.TagSettings["FOREIGNKEY"]]
			referencesKey, ok2 := mainSchema.FieldsByName[relation.Field.TagSettings["REFERENCES"]]
			var hasOn bool
			if ok1 && ok2 {
				foreignKeyName := foreignKey.DBName
				referencesKeyName := referencesKey.DBName
				if len(foreignKeyName) > 0 && len(referencesKeyName) > 0 {
					mainAlias := r.Model.TableName()
					if len(r.Model.Alias()) > 0 {
						mainAlias = r.Model.Alias()
					}
					alias := v.Model.TableName()
					if len(v.Model.Alias()) > 0 {
						alias = v.Model.Alias()
					}
					build.WriteString(fmt.Sprintf("ON %s.%s = %s.%s ", mainAlias, referencesKeyName, alias, foreignKeyName))
					hasOn = true
				}
			}
			if len(v.ON) > 0 {
				if hasOn {
					build.WriteString(fmt.Sprintf("AND %s ", v.ON[0]))
				} else {
					build.WriteString(fmt.Sprintf("ON %s ", v.ON[0]))
				}
			}
			var selectFields []string
			if len(v.Fields) == 0 {
				selectFields = q.GetHasOneJoinSelectFields(v.Model, v.As)
			} else {
				selectFields = v.Fields
			}
			return func(db *gorm.DB) *gorm.DB {
				db = db.Select(db.Statement.Selects, q.GetSelectFields(r.Model))
				db = db.Select(db.Statement.Selects, selectFields)
				if len(v.ON) > 1 {
					return db.Joins(build.String(), v.ON[1:])
				}
				return db.Joins(build.String())
			}
		}
		return func(db *gorm.DB) *gorm.DB {
			if len(r.ON) > 0 {
				if len(r.ON) > 1 {
					db.Where(r.ON[0], r.ON[1:])
				} else {
					db.Where(r.ON[0])
				}
			}
			if len(r.Fields) > 0 {
				db.Select(db.Statement.Selects, r.Fields)
			}
			if len(r.OrderBy) > 0 {
				db.Order(r.OrderBy)
			}
			if len(r.GroupBy) > 0 {
				db.Group(r.GroupBy)
			}
			subFunc := func(db *gorm.DB) *gorm.DB {
				if len(v.ON) > 0 {
					if len(v.ON) > 1 {
						db.Where(v.ON[0], v.ON[1:])
					} else {
						db.Where(v.ON[0])
					}
				}
				if len(v.Fields) > 0 {
					db.Select(db.Statement.Selects, v.Fields)
				}
				if len(v.OrderBy) > 0 {
					db.Order(v.OrderBy)
				}
				if len(v.GroupBy) > 0 {
					db.Group(v.GroupBy)
				}
				return db
			}
			return db.Preload(v.As, subFunc)
		}
	}
	return func(db *gorm.DB) *gorm.DB {
		return db
	}
}

func (q *QueryBuilder[T]) handleHasOneRelations(db *gorm.DB, mainModel Model, relations []Relation) *gorm.DB {
	for _, v := range relations {
		db, isHasOne := q.buildHasOneSql(db, mainModel, v)
		if len(v.Relations) > 0 && isHasOne {
			db = q.handleHasOneSubRelations(db, v.Model, v.Relations)
		}
	}
	return db
}

func (q *QueryBuilder[T]) handleHasOneSubRelations(db *gorm.DB, mainModel Model, relations []SubRelation) *gorm.DB {
	for _, v := range relations {
		db = q.buildHasOneSubSql(db, mainModel, v)
	}
	return db
}

func (q *QueryBuilder[T]) buildHasOneSql(db *gorm.DB, mainModel Model, v Relation) (*gorm.DB, bool) {
	var build strings.Builder
	mainSchema, err := schema.Parse(mainModel, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return db, false
	}
	relation, ok := mainSchema.Relationships.Relations[v.As]
	if !ok {
		return db, false
	}
	if relation.Type != schema.HasOne {
		return db, false
	}
	if v.Required {
		build.WriteString("INNER JOIN ")
	} else {
		build.WriteString("LEFT JOIN ")
	}
	build.WriteString(fmt.Sprintf("`%s` ", v.Model.TableName()))
	if len(v.Model.Alias()) > 0 {
		build.WriteString(fmt.Sprintf("`%s` ", v.Model.Alias()))
	} else {
		build.WriteString(fmt.Sprintf("`%s` ", v.Model.TableName()))
	}
	subSchema, err := schema.Parse(v.Model, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return db, false
	}
	foreignKey, ok1 := subSchema.FieldsByName[relation.Field.TagSettings["FOREIGNKEY"]]
	referencesKey, ok2 := mainSchema.FieldsByName[relation.Field.TagSettings["REFERENCES"]]
	var hasOn bool
	if ok1 && ok2 {
		foreignKeyName := foreignKey.DBName
		referencesKeyName := referencesKey.DBName
		if len(foreignKeyName) > 0 && len(referencesKeyName) > 0 {
			mainAlias := mainModel.TableName()
			if len(mainModel.Alias()) > 0 {
				mainAlias = mainModel.Alias()
			}
			alias := v.Model.TableName()
			if len(v.Model.Alias()) > 0 {
				alias = v.Model.Alias()
			}
			build.WriteString(fmt.Sprintf("ON %s.%s = %s.%s ", mainAlias, referencesKeyName, alias, foreignKeyName))
			hasOn = true
		}
	}
	if len(v.ON) > 0 {
		if hasOn {
			build.WriteString(fmt.Sprintf("AND %s ", v.ON[0]))
		} else {
			build.WriteString(fmt.Sprintf("ON %s ", v.ON[0]))
		}
	}
	if len(v.ON) > 1 {
		db = db.Joins(build.String(), v.ON[1:])
	} else {
		db = db.Joins(build.String())
	}
	if len(v.Fields) == 0 {
		db = db.Select(db.Statement.Selects, q.GetHasOneJoinSelectFields(v.Model, v.As))
	} else {
		db = db.Select(db.Statement.Selects, v.Fields)
	}
	return db, true
}

func (q *QueryBuilder[T]) buildHasOneSubSql(db *gorm.DB, mainModel Model, v SubRelation) *gorm.DB {
	var build strings.Builder
	mainSchema, err := schema.Parse(mainModel, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return db
	}
	relation, ok := mainSchema.Relationships.Relations[v.As]
	if !ok {
		return db
	}
	if relation.Type != schema.HasOne {
		return db
	}
	if v.Required {
		build.WriteString("INNER JOIN ")
	} else {
		build.WriteString("LEFT JOIN ")
	}
	build.WriteString(fmt.Sprintf("`%s` ", v.Model.TableName()))
	if len(v.Model.Alias()) > 0 {
		build.WriteString(fmt.Sprintf("`%s` ", v.Model.Alias()))
	} else {
		build.WriteString(fmt.Sprintf("`%s` ", v.Model.TableName()))
	}
	subSchema, err := schema.Parse(v.Model, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return db
	}
	foreignKey, ok1 := subSchema.FieldsByName[relation.Field.TagSettings["FOREIGNKEY"]]
	referencesKey, ok2 := mainSchema.FieldsByName[relation.Field.TagSettings["REFERENCES"]]
	var hasOn bool
	if ok1 && ok2 {
		foreignKeyName := foreignKey.DBName
		referencesKeyName := referencesKey.DBName
		if len(foreignKeyName) > 0 && len(referencesKeyName) > 0 {
			mainAlias := mainModel.TableName()
			if len(mainModel.Alias()) > 0 {
				mainAlias = mainModel.Alias()
			}
			alias := v.Model.TableName()
			if len(v.Model.Alias()) > 0 {
				alias = v.Model.Alias()
			}
			build.WriteString(fmt.Sprintf("ON %s.%s = %s.%s ", mainAlias, referencesKeyName, alias, foreignKeyName))
			hasOn = true
		}
	}
	if len(v.ON) > 0 {
		if hasOn {
			build.WriteString(fmt.Sprintf("AND %s ", v.ON[0]))
		} else {
			build.WriteString(fmt.Sprintf("ON %s ", v.ON[0]))
		}
	}
	if len(v.ON) > 1 {
		db = db.Joins(build.String(), v.ON[1:])
	} else {
		db = db.Joins(build.String())
	}
	if len(v.Fields) == 0 {
		db = db.Select(db.Statement.Selects, q.GetHasOneJoinSelectFields(v.Model, v.As))
	} else {
		db = db.Select(db.Statement.Selects, v.Fields)
	}
	return db
}

func (q *QueryBuilder[T]) fillConditions(db *gorm.DB, conditions []map[string]interface{}) *gorm.DB {
	for _, v := range conditions {
		key := v["key"]
		alias := q.Model.TableName()
		if len(q.Model.Alias()) > 0 {
			alias = q.Model.Alias()
		}
		if !strings.Contains(key.(string), ".") {
			key = fmt.Sprintf("%s.%s", alias, key)
		}
		if _, ok := v["val"]; ok {
			db.Where(key, v["val"])
		} else {
			db.Where(key)
		}
	}
	return db
}

func (q *QueryBuilder[T]) GetSelectFields(model Model) []string {
	result := make([]string, 0)
	schema, _ := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	for _, v := range schema.Fields {
		if _, ok := v.TagSettings[NoSelect]; ok {
			continue
		}
		if _, ok := v.TagSettings["-"]; ok {
			continue
		}
		alias := model.TableName()
		if len(model.Alias()) > 0 {
			alias = model.Alias()
		}
		result = append(result, fmt.Sprintf("`%s`.`%s`", alias, v.DBName))
	}
	return result
}

func (q *QueryBuilder[T]) GetHasOneJoinSelectFields(model Model, as string) []string {
	result := make([]string, 0)
	schema, _ := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	for _, v := range schema.Fields {
		if _, ok := v.TagSettings[NoSelect]; ok {
			continue
		}
		if _, ok := v.TagSettings["-"]; ok {
			continue
		}
		alias := model.TableName()
		if len(model.Alias()) > 0 {
			alias = model.Alias()
		}
		result = append(result, fmt.Sprintf("`%s`.`%s` AS `%s__%s`", alias, v.DBName, as, v.DBName))
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
		logger: log.New(app.loggerConfig.LoggerPath, app.serverConfig.AppName+"_sql.log", app.loggerConfig.LoggerLevel,
			app.loggerConfig.LoggerMaxAge),
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
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&loc=Local", app.ormConfig.UserName, app.ormConfig.Password,
			app.ormConfig.Host, app.ormConfig.Port, app.ormConfig.DbName)
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
