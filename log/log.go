package log

import (
	"bytes"
	"os"
	"path"
	"strings"
	"time"

	"github.com/funswe/flow/utils/files"
	"github.com/funswe/flow/utils/json"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Entry
}

// MyFormatter 定义日志格式化
type MyFormatter struct {
}

func New(logPath, logFileName, loggerLevel string, maxAge int64) *Logger {
	if !files.PathExists(logPath) {
		os.MkdirAll(logPath, os.ModePerm)
	}
	if maxAge == 0 {
		maxAge = 30
	}
	baseLogPath := path.Join(logPath, logFileName)
	writer, _ := rotatelogs.New(
		baseLogPath+".%Y-%m-%d",
		rotatelogs.WithLinkName(baseLogPath),
		rotatelogs.WithMaxAge(time.Duration(maxAge)*24*time.Hour),
	)
	format := new(MyFormatter)
	lfHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: writer,
		logrus.InfoLevel:  writer,
		logrus.WarnLevel:  writer,
		logrus.ErrorLevel: writer,
		logrus.FatalLevel: writer,
		logrus.PanicLevel: writer,
	}, format)
	logger := logrus.New()
	logger.SetFormatter(format)
	logger.AddHook(lfHook)
	level, err := logrus.ParseLevel(loggerLevel)
	if err != nil {
		logrus.Fatal(err)
	}
	logger.SetLevel(level)
	return &Logger{&logrus.Entry{Logger: logger}}
}

func (l *Logger) Create(fields logrus.Fields) *Logger {
	e := l.WithFields(fields)
	return &Logger{e}
}

// Format 定义日志格式
func (f *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	logTime := entry.Time.In(time.Local).Format("2006-01-02 15:04:05.000")
	b.WriteString(logTime)
	b.WriteByte('[')
	logLevel := entry.Level.String()
	b.WriteString(strings.ToUpper(logLevel))
	b.WriteByte(']')
	b.WriteByte(' ')
	logMsg := entry.Message
	b.WriteString(logMsg)
	if len(entry.Data) > 0 {
		logData, _ := json.Marshal(entry.Data)
		b.WriteByte(' ')
		b.Write(logData)
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}
