package flow

import (
	"testing"

	"gorm.io/gorm/logger"
)

// 默认数据库配置与连接池配置
func TestDefaultOrmConfigs(t *testing.T) {
	config := defOrmConfig()
	if config == nil {
		t.Fatal("默认数据库配置不应为nil")
	}
	if config.Pool == nil {
		t.Fatal("默认连接池配置不应为nil")
	}
	pool := defOrmPool()
	if pool == nil || pool.MaxIdle <= 0 || pool.MaxOpen <= 0 {
		t.Fatalf("默认连接池配置不合法：%+v", pool)
	}
	if pool.MaxOpen < pool.MaxIdle {
		t.Fatalf("最大连接数不应小于空闲连接数：%+v", pool)
	}
}

// 数据库日志适配器：可切换日志级别
func TestOrmLoggerLogMode(t *testing.T) {
	dbLog := defOrmLogger()
	if dbLog == nil {
		t.Fatal("默认数据库日志不应为nil")
	}
	if got := dbLog.LogMode(logger.Info); got == nil {
		t.Fatal("LogMode应返回日志接口")
	}
	if got := dbLog.LogMode(logger.Silent); got == nil {
		t.Fatal("LogMode应返回日志接口")
	}
}
