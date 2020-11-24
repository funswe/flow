package flow

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"time"

	"github.com/funswe/flow/log"
	"gorm.io/driver/mysql"
)

// 定义数据库配置
type OrmConfig struct {
	Enable   bool
	UserName string
	Password string
	DbName   string
	Host     string
	Port     int
	Pool     *OrmPool
}

// 定义数据库连接池配置
type OrmPool struct {
	MaxIdle         int
	MaxOpen         int
	ConnMaxLifeTime int64
	ConnMaxIdleTime int64
}

// 定义数据库查询条件操作符
type OrmOp struct {
	Eq   string
	Gt   string
	Gte  string
	Lt   string
	Lte  string
	IN   string
	Like string
	Or   string
}

// 定义数据库JOIN表类型
type OrmJoinType struct {
	CrossJoin string
	InnerJoin string
	LeftJoin  string
	RightJoin string
}

// 定义数据库表列结构
type OrmColumn struct {
	Table  string // 表名
	Column string // 列名
	Alias  string // 别名
}

// 定义数据库查询表结构
type OrmFromTable struct {
	Table string     // 表名
	Alias string     // 别名
	Joins []*OrmJoin // JOIN表集合
}

// 定义数据库JOIN表结构
type OrmJoin struct {
	Type  string      // JOIN类型
	Table string      // 表名
	Alias string      // 别名
	ON    []*OrmWhere // JOIN条件
}

// 定义数据库条件结构
type OrmWhere struct {
	Column *OrmColumn  // 表列
	Opt    string      // 操作符
	Value  interface{} //条件的值
}

// 定义数据库ORDER BY结构
type OrmOrderBy struct {
	Column *OrmColumn // 表列
	Desc   bool       // 是否是倒序
}

// 定义数据库LIMIT结构
type OrmLimit struct {
	Limit  int
	Offset int
}

// 定义数据库GROUP BY结构
type OrmGroupBy struct {
	Columns []*OrmColumn // 列集合
	Having  []*OrmWhere  // having条件集合
}

// 定义数据库操作对象
type Orm struct {
	app      *Application // app对象
	db       *gorm.DB     // grom db对象
	Op       *OrmOp       // 数据库查询条件操作符对象
	JoinType *OrmJoinType // 数据库JOIN表类型对象
}

// 定义查询字符段的结构体
type Fields struct {
	fields []*OrmColumn
}

// 添加查询字段，可以链式处理
func (f *Fields) Add(column string, table ...string) *Fields {
	if len(column) == 0 {
		return f
	}
	c := &OrmColumn{
		Column: column,
	}
	if len(table) >= 2 {
		c.Table = table[0]
		c.Alias = table[1]
	} else if len(table) == 1 {
		c.Table = table[0]
	}
	f.fields = append(f.fields, c)
	return f
}

func (f *Fields) Get() []*OrmColumn {
	return f.fields
}

type Conditions struct {
	conditions []*OrmWhere
}

// 添加查询条件，可以链式处理
func (c *Conditions) Add(columnName, op string, val interface{}, table ...string) *Conditions {
	if len(columnName) == 0 {
		return c
	}
	column := &OrmColumn{
		Column: columnName,
	}
	if len(table) >= 2 {
		column.Table = table[0]
		column.Alias = table[1]
	} else if len(table) == 1 {
		column.Table = table[0]
	}
	w := &OrmWhere{
		Column: column,
		Opt:    op,
		Value:  val,
	}
	c.conditions = append(c.conditions, w)
	return c
}

func (c *Conditions) Get() []*OrmWhere {
	return c.conditions
}

func (orm *Orm) NewConditions() *Conditions {
	return &Conditions{conditions: make([]*OrmWhere, 0)}
}

func (orm *Orm) NewFields() *Fields {
	return &Fields{fields: make([]*OrmColumn, 0)}
}

// 返回grom db对象，用于原生查询使用
func (orm *Orm) DB() *gorm.DB {
	if orm.db == nil {
		panic(errors.New("no db server available"))
	}
	return orm.db
}

// 数据库查询方法
func (orm *Orm) Query(dest interface{}, fields interface{}, fromTable *OrmFromTable, conditions []*OrmWhere, orderBy []*OrmOrderBy, limit *OrmLimit, groupBy *OrmGroupBy) error {
	if orm.db == nil {
		panic(errors.New("no db server available"))
	}
	stmt := gorm.Statement{DB: orm.db, Clauses: map[string]clause.Clause{}}
	buildName := make([]string, 0)
	//if len(fields) > 0 {
	stmt.AddClause(orm.buildFields(fields))
	buildName = append(buildName, "SELECT")
	//}
	if fromTable != nil {
		stmt.AddClause(orm.buildFromTable(fromTable))
		buildName = append(buildName, "FROM")
	}
	var params map[string]interface{}
	var where clause.Where
	if len(conditions) > 0 {
		where, params = orm.buildConditions(conditions)
		stmt.AddClause(where)
		buildName = append(buildName, "WHERE")
	}
	if len(orderBy) > 0 {
		stmt.AddClause(orm.buildOrderBy(orderBy))
		buildName = append(buildName, "ORDER BY")
	}
	if groupBy != nil {
		stmt.AddClause(orm.buildGroupBy(groupBy))
		buildName = append(buildName, "GROUP BY")
	}
	if limit != nil {
		stmt.AddClause(orm.buildLimit(limit))
		buildName = append(buildName, "LIMIT")
	}
	stmt.Build(buildName...)
	sql := stmt.SQL.String()
	result := orm.db.Raw(sql, params).Scan(dest)
	return result.Error
}

// 返回查询的总条数
func (orm *Orm) Count(count *int64, fromTable *OrmFromTable, conditions []*OrmWhere) error {
	if orm.db == nil {
		panic(errors.New("no db server available"))
	}
	stmt := gorm.Statement{DB: orm.db, Clauses: map[string]clause.Clause{}}
	buildName := make([]string, 0)
	stmt.AddClause(clause.Select{
		Expression: clause.Expr{SQL: "count(1)"},
	})
	buildName = append(buildName, "SELECT")
	if fromTable != nil {
		stmt.AddClause(orm.buildFromTable(fromTable))
		buildName = append(buildName, "FROM")
	}
	var params map[string]interface{}
	var where clause.Where
	if len(conditions) > 0 {
		where, params = orm.buildConditions(conditions)
		stmt.AddClause(where)
		buildName = append(buildName, "WHERE")
	}
	stmt.Build(buildName...)
	sql := stmt.SQL.String()
	result := orm.db.Raw(sql, params).Scan(count)
	return result.Error
}

// 构建查询的列信息
func (orm *Orm) buildFields(fields interface{}) clause.Select {
	if fields == nil {
		return clause.Select{}
	}
	columns := make([]clause.Column, 0)
	switch fs := fields.(type) {
	case []*OrmColumn:
		for _, field := range fs {
			columns = append(columns, clause.Column{
				Table: field.Table,
				Name:  field.Column,
				Alias: field.Alias,
			})
			//if len(field.Table) > 0 {
			//	columns = append(columns, clause.Column{
			//		Table: field.Table,
			//		Name:  field.Column,
			//		Alias: field.Alias,
			//	})
			//} else {
			//	sql := field.Column
			//	if len(field.Alias) > 0 {
			//		sql = fmt.Sprintf("%s as %s", field.Column, field.Alias)
			//	}
			//	return clause.Select{
			//		Expression: clause.Expr{SQL: sql},
			//	}
			//}
		}
	case []string:
		for _, field := range fs {
			columns = append(columns, clause.Column{
				Name: field,
			})
		}
	default:
		panic(errors.New("invalid fields type"))
	}
	return clause.Select{
		Columns: columns,
	}
}

// 构建查询的from表
func (orm *Orm) buildFromTable(fromTable *OrmFromTable) clause.From {
	from := clause.From{
		Tables: []clause.Table{
			{
				Name: fromTable.Table, Alias: fromTable.Alias,
			},
		},
	}
	if len(fromTable.Joins) > 0 {
		joins := make([]clause.Join, 0)
		for _, join := range fromTable.Joins {
			j := clause.Join{
				Type: clause.JoinType(join.Type),
				Table: clause.Table{
					Name:  join.Table,
					Alias: join.Alias,
				},
			}
			if len(join.ON) > 0 {
				j.ON = clause.Where{
					Exprs: orm.parseWhere(join.ON),
				}
			}
			joins = append(joins, j)
		}
		from.Joins = joins
	}
	return from
}

// 构建查询的条件
func (orm *Orm) buildConditions(conditions []*OrmWhere) (clause.Where, map[string]interface{}) {
	params := make(map[string]interface{}, 0)
	where := clause.Where{
		Exprs: orm.parseConditionsWhere(conditions, params),
	}
	return where, params
}

// 构建查询的ORDER BY
func (orm *Orm) buildOrderBy(orderBy []*OrmOrderBy) clause.OrderBy {
	columns := make([]clause.OrderByColumn, 0)
	for _, order := range orderBy {
		columns = append(columns, clause.OrderByColumn{
			Column: clause.Column{
				Table: order.Column.Table,
				Name:  order.Column.Column,
				Alias: order.Column.Alias,
			},
			Desc: order.Desc,
		})
	}
	return clause.OrderBy{
		Columns: columns,
	}
}

// 构建查询的GROUP BY
func (orm *Orm) buildGroupBy(groupBy *OrmGroupBy) clause.GroupBy {
	result := clause.GroupBy{}
	columns := groupBy.Columns
	if len(columns) > 0 {
		resultColumns := make([]clause.Column, 0)
		for _, column := range columns {
			resultColumns = append(resultColumns, clause.Column{
				Table: column.Table,
				Name:  column.Column,
				Alias: column.Alias,
			})
		}
		result.Columns = resultColumns
	}
	having := groupBy.Having
	if len(having) > 0 {
		result.Having = orm.parseWhere(groupBy.Having)
	}
	return result
}

// 构建查询的LIMIT信息
func (orm *Orm) buildLimit(limit *OrmLimit) clause.Limit {
	return clause.Limit{
		Limit:  limit.Limit,
		Offset: limit.Offset,
	}
}

// 解析条件对象信息
func (orm *Orm) parseWhere(wheres []*OrmWhere) []clause.Expression {
	onExprs := make([]clause.Expression, 0)
	for _, o := range wheres {
		var value interface{}
		var values []interface{}
		var valueWheres []*OrmWhere
		switch t := o.Value.(type) {
		case *OrmColumn:
			value = clause.Column{Table: t.Table, Name: t.Column, Alias: t.Alias}
		case []interface{}:
			values = t
		case []*OrmWhere:
			valueWheres = t
		default:
			value = t
		}
		switch o.Opt {
		case orm.Op.Eq:
			onExprs = append(onExprs, clause.Eq{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: value,
			})
		case orm.Op.Gt:
			onExprs = append(onExprs, clause.Gt{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: value,
			})
		case orm.Op.Gte:
			onExprs = append(onExprs, clause.Gte{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: value,
			})
		case orm.Op.Lt:
			onExprs = append(onExprs, clause.Lt{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: value,
			})
		case orm.Op.Lte:
			onExprs = append(onExprs, clause.Lte{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: value,
			})
		case orm.Op.IN:
			onExprs = append(onExprs, clause.IN{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Values: values,
			})
		case orm.Op.Like:
			onExprs = append(onExprs, clause.Like{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: value,
			})
		case orm.Op.Or:
			onExprs = append(onExprs, clause.OrConditions{
				Exprs: orm.parseWhere(valueWheres),
			})
		}
	}
	return onExprs
}

// 解析条件对象信息，包含条件的值
func (orm *Orm) parseConditionsWhere(wheres []*OrmWhere, params map[string]interface{}) []clause.Expression {
	onExprs := make([]clause.Expression, 0)
	for _, o := range wheres {
		var value interface{}
		var values []interface{}
		var valueWheres []*OrmWhere
		switch t := o.Value.(type) {
		case []*OrmWhere:
			valueWheres = t
		case []interface{}:
			values = t
		default:
			value = t
		}
		paramsKey := fmt.Sprintf("%s:%s", o.Column.Table, o.Column.Column)
		switch o.Opt {
		case orm.Op.Eq:
			params[paramsKey] = value
			onExprs = append(onExprs, clause.Eq{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: clause.Expr{
					SQL: fmt.Sprintf("@%s", paramsKey),
				},
			})
		case orm.Op.Gt:
			params[paramsKey] = value
			onExprs = append(onExprs, clause.Gt{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: clause.Expr{
					SQL: fmt.Sprintf("@%s", paramsKey),
				},
			})
		case orm.Op.Gte:
			params[paramsKey] = value
			onExprs = append(onExprs, clause.Gte{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: clause.Expr{
					SQL: fmt.Sprintf("@%s", paramsKey),
				},
			})
		case orm.Op.Lt:
			params[paramsKey] = value
			onExprs = append(onExprs, clause.Lt{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: clause.Expr{
					SQL: fmt.Sprintf("@%s", paramsKey),
				},
			})
		case orm.Op.Lte:
			params[paramsKey] = value
			onExprs = append(onExprs, clause.Lte{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: clause.Expr{
					SQL: fmt.Sprintf("@%s", paramsKey),
				},
			})
		case orm.Op.IN:
			vv := make([]interface{}, 0)
			for i, v := range values {
				pKey := fmt.Sprintf("%s:%s:%d", o.Column.Table, o.Column.Column, i)
				params[pKey] = v
				vv = append(vv, clause.Expr{
					SQL: fmt.Sprintf("@%s", pKey),
				})
			}
			onExprs = append(onExprs, clause.IN{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Values: vv,
			})
		case orm.Op.Like:
			params[paramsKey] = value
			onExprs = append(onExprs, clause.Like{
				Column: clause.Column{Table: o.Column.Table, Name: o.Column.Column, Alias: o.Column.Alias}, Value: clause.Expr{
					SQL: fmt.Sprintf("@%s", paramsKey),
				},
			})
		case orm.Op.Or:
			onExprs = append(onExprs, clause.OrConditions{
				Exprs: orm.parseConditionsWhere(valueWheres, params),
			})
		}
	}
	return onExprs
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
	return &Orm{Op: newOp(), JoinType: newJoinType()}
}

// 返回数据库操作符对象
func newOp() *OrmOp {
	return &OrmOp{
		Eq:   "eq",
		Gt:   "gt",
		Gte:  "gte",
		Lt:   "lt",
		Lte:  "lte",
		IN:   "in",
		Like: "like",
		Or:   "or",
	}
}

// 返回数据库JOIN类型对象
func newJoinType() *OrmJoinType {
	return &OrmJoinType{
		CrossJoin: "CROSS",
		InnerJoin: "INNER",
		LeftJoin:  "LEFT",
		RightJoin: "RIGHT",
	}
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
		app.orm.db = db
		logFactory.Info("db server init ok")
	}
}
