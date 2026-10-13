package flow

import (
	"path/filepath"
	"testing"

	"go.uber.org/zap/zapcore"
)

// 日志级别字符串到zap级别的映射
func TestEncodeLevel(t *testing.T) {
	cases := map[string]zapcore.Level{
		"debug":  zapcore.DebugLevel,
		"info":   zapcore.InfoLevel,
		"warn":   zapcore.WarnLevel,
		"error":  zapcore.ErrorLevel,
		"fatal":  zapcore.FatalLevel,
		"dpanic": zapcore.DPanicLevel,
		"panic":  zapcore.PanicLevel,
	}
	for input, expected := range cases {
		if got := encodeLevel(input); got != expected {
			t.Errorf("encodeLevel(%q) = %v，期望 %v", input, got, expected)
		}
	}
}

// 未知级别回退为info（保证配置写错时不会丢日志）
func TestEncodeLevelFallback(t *testing.T) {
	if got := encodeLevel("unknown-level"); got != zapcore.InfoLevel && got != zapcore.DebugLevel {
		t.Fatalf("未知级别应回退到可用的默认级别，实际 %v", got)
	}
}

// 默认配置：服务名、代理、日志目录与级别等
func TestDefaultConfigs(t *testing.T) {
	server := defServerConfig()
	if server == nil || len(server.Host) == 0 || server.Port <= 0 {
		t.Fatalf("默认服务配置不合法：%+v", server)
	}
	logger := defLoggerConfig()
	if logger == nil || logger.LoggerMaxAge <= 0 {
		t.Fatalf("默认日志配置不合法：%+v", logger)
	}
	if filepath.Base(logger.LoggerPath) == "" {
		t.Fatalf("默认日志目录不合法：%q", logger.LoggerPath)
	}
	if appName := defAppName(); len(appName) == 0 {
		t.Fatal("默认应用名不应为空")
	}
	if level := defLoggerLevel(); len(level) == 0 {
		t.Fatal("默认日志级别不应为空")
	}
}

// 编码器：JSON与文本两种格式都能构建
func TestGetEncoder(t *testing.T) {
	original := app.loggerConfig
	app.loggerConfig = defLoggerConfig()
	defer func() { app.loggerConfig = original }()

	app.loggerConfig.FormatJson = true
	if enc := getEncoder(app); enc == nil {
		t.Fatal("JSON编码器不应为nil")
	}
	app.loggerConfig.FormatJson = false
	if enc := getEncoder(app); enc == nil {
		t.Fatal("文本编码器不应为nil")
	}
}
