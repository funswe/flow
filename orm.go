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

type OrmJoinType struct {
	CrossJoin string
	InnerJoin string
	LeftJoin  string
	RightJoin string
}

type OrmColumn struct {
	Table  string
	Column string
	Alias  string
}

type OrmFromTable struct {
	Table string
	Alias string
	Joins []*OrmJoin
}

type OrmJoin struct {
	Type  string
	Table string
	Alias string
	ON    []*OrmWhere
}

type OrmWhere struct {
	Column *OrmColumn
	Opt    string
	Value  interface{}
}

type OrmOrderBy struct {
	Column *OrmColumn
	Desc   bool
}

type OrmLimit struct {
	Limit  int
	Offset int
}

type OrmGroupBy struct {
	Columns []*OrmColumn
	Having  []*OrmWhere
}

type Orm struct {
	app      *Application
	db       *gorm.DB
	Op       *OrmOp
	JoinType *OrmJoinType
}

func (orm *Orm) DB() *gorm.DB {
	if orm.db == nil {
		panic(errors.New("no db server available"))
	}
	return orm.db
}

func (orm *Orm) Query(dest interface{}, fields []*OrmColumn, fromTable *OrmFromTable, conditions []*OrmWhere, orderBy []*OrmOrderBy, limit *OrmLimit, groupBy *OrmGroupBy) error {
	if orm.db == nil {
		panic(errors.New("no db server available"))
	}
	stmt := gorm.Statement{DB: orm.db, Clauses: map[string]clause.Clause{}}
	buildName := make([]string, 0)
	if len(fields) > 0 {
		stmt.AddClause(orm.buildFields(fields))
		buildName = append(buildName, "SELECT")
	}
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

func (orm *Orm) buildFields(fields []*OrmColumn) clause.Select {
	columns := make([]clause.Column, 0)
	for _, field := range fields {
		if len(field.Table) > 0 {
			columns = append(columns, clause.Column{
				Table: field.Table,
				Name:  field.Column,
				Alias: field.Alias,
			})
		} else {
			sql := field.Column
			if len(field.Alias) > 0 {
				sql = fmt.Sprintf("%s as %s", field.Column, field.Alias)
			}
			return clause.Select{
				Expression: clause.Expr{SQL: sql},
			}
		}
	}
	return clause.Select{
		Columns: columns,
	}
}

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

func (orm *Orm) buildConditions(conditions []*OrmWhere) (clause.Where, map[string]interface{}) {
	params := make(map[string]interface{}, 0)
	where := clause.Where{
		Exprs: orm.parseConditionsWhere(conditions, params),
	}
	return where, params
}

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

func (orm *Orm) buildLimit(limit *OrmLimit) clause.Limit {
	return clause.Limit{
		Limit:  limit.Limit,
		Offset: limit.Offset,
	}
}

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

func defOrm() *Orm {
	return &Orm{Op: newOp(), JoinType: newJoinType()}
}

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

func newJoinType() *OrmJoinType {
	return &OrmJoinType{
		CrossJoin: "CROSS",
		InnerJoin: "INNER",
		LeftJoin:  "LEFT",
		RightJoin: "RIGHT",
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
		app.orm.db = db
		logFactory.Info("db server init ok")
	}
}
