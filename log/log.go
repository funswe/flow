package log

import (
	"fmt"
	"github.com/funswe/flow/utils/json"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

type Log struct {
	*logrus.Entry
}

func New(logPath, logFileName string) *Log {
	if !pathExists(logPath) {
		os.MkdirAll(logPath, os.ModePerm)
	}
	baseLogPaht := path.Join(logPath, logFileName)
	writer, err := rotatelogs.New(
		baseLogPaht+".%Y-%m-%d",
		rotatelogs.WithLinkName(baseLogPaht),
	)
	if err != nil {
		logrus.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}
	lfHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: writer,
		logrus.InfoLevel:  writer,
		logrus.WarnLevel:  writer,
		logrus.ErrorLevel: writer,
		logrus.FatalLevel: writer,
		logrus.PanicLevel: writer,
	}, &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logger := logrus.New()
	logger.SetFormatter(new(MyJSONFormatter))
	logger.AddHook(lfHook)
	log := &Log{&logrus.Entry{}}
	log.Logger = logger
	return log
}

func (l *Log) Create(fields logrus.Fields) *Log {
	e := l.WithFields(fields)
	return &Log{
		e,
	}
}

type MyJSONFormatter struct {
}

func (f *MyJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Note this doesn't include Time, Level and Message which are available on
	// the Entry. Consult `godoc` on information about those fields or read the
	// source of the official loggers.
	fmt.Println(entry)
	serialized, err := json.Marshal(entry.Data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
	}
	return append(serialized, '\n'), nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}
