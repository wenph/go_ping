package utils

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"path/filepath"
	"strings"
)

// Log 是应用程序中使用的全局日志实例。
var Log = logrus.New()

func init() {
	// 设置日志格式，例如 JSON
	Log.SetFormatter(&CustomFormatter{
		TimestampFormat: "2006-01-02 15:04:05", // 你希望的时间戳格式
	})
	// 设置日志级别
	Log.SetLevel(logrus.InfoLevel)
	// 创建一个新的 lumberjack.Logger 实例
	logFile := &lumberjack.Logger{
		Filename:   "/tmp/go_ping.log", // 日志文件路径
		MaxSize:    100,                // 文件最大大小（MB）
		MaxBackups: 10,                 // 保留旧文件的最大个数
		MaxAge:     30,                 // 保留旧文件的最大天数
		Compress:   true,               // 是否压缩/归档旧文件
	}
	Log.SetOutput(logFile)
	// 设置是否显示时间戳、是否打印调用方法等
	Log.SetReportCaller(true)
}

// CustomFormatter 自定义日志格式
type CustomFormatter struct {
	TimestampFormat string
}

// Format 实现 logrus.Formatter 接口的 Format 方法
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("%s %s %s:%d %s\n",
		entry.Time.Format(f.TimestampFormat),
		strings.ToUpper(entry.Level.String()),
		filepath.Base(entry.Caller.File),
		entry.Caller.Line,
		entry.Message)), nil
}

func SetLogLevel(logLevel string) {
	if logLevel == "error" {
		Log.SetLevel(logrus.ErrorLevel)
	} else if logLevel == "warn" {
		Log.SetLevel(logrus.WarnLevel)
	} else if logLevel == "info" {
		Log.SetLevel(logrus.InfoLevel)
	} else if logLevel == "debug" {
		Log.SetLevel(logrus.DebugLevel)
	}
}
