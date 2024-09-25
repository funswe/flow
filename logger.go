package flow

import (
	"errors"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

func getLogger(app *Application, initialFields map[string]interface{}) *zap.Logger {
	writeSyncer := getLogWriter(app)
	encoder := getEncoder(app)
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(writeSyncer), encodeLevel(strings.ToLower(app.loggerConfig.LoggerLevel))),
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
	)
	options := make([]zap.Option, 0)
	if len(initialFields) > 0 {
		fields := make([]zap.Field, 0, len(initialFields))
		for k, v := range initialFields {
			fields = append(fields, zap.Any(k, v))
		}
		options = append(options, zap.Fields(fields...))
	}
	return zap.New(core, options...)
}

func getOrmLogger(app *Application, initialFields map[string]interface{}) *zap.Logger {
	writeSyncer := getOrmLogWriter(app)
	encoder := getEncoder(app)
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(writeSyncer), encodeLevel(strings.ToLower(app.loggerConfig.LoggerLevel))),
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
	)
	options := make([]zap.Option, 0)
	if len(initialFields) > 0 {
		fields := make([]zap.Field, 0, len(initialFields))
		for k, v := range initialFields {
			fields = append(fields, zap.Any(k, v))
		}
		options = append(options, zap.Fields(fields...))
	}
	return zap.New(core, options...)
}

func getEncoder(app *Application) zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
	}
	if app.loggerConfig.FormatJson {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter(app *Application) io.Writer {
	baseLogPath := path.Join(app.loggerConfig.LoggerPath, app.serverConfig.AppName)
	writer, err := rotatelogs.New(
		baseLogPath+"_%Y-%m-%d.log",
		rotatelogs.WithLinkName(baseLogPath),
		rotatelogs.WithMaxAge(time.Duration(app.loggerConfig.LoggerMaxAge)*24*time.Hour),
	)
	if err != nil {
		panic(errors.New("init logger failed: " + err.Error()))
	}
	return writer
}

func getOrmLogWriter(app *Application) io.Writer {
	baseLogPath := path.Join(app.loggerConfig.LoggerPath, app.serverConfig.AppName)
	writer, err := rotatelogs.New(
		baseLogPath+"_sql_%Y-%m-%d.log",
		rotatelogs.WithLinkName(baseLogPath),
		rotatelogs.WithMaxAge(time.Duration(app.loggerConfig.LoggerMaxAge)*24*time.Hour),
	)
	if err != nil {
		panic(errors.New("init orm logger failed: " + err.Error()))
	}
	return writer
}

func customTimeEncoder(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func encodeLevel(level string) zapcore.Level {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	case "dpanic":
		zapLevel = zapcore.DPanicLevel
	case "panic":
		zapLevel = zapcore.PanicLevel
	case "fatal":
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	return zapLevel
}
