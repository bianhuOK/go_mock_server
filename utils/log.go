package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// CustomFormatter 自定义日志格式
type CustomFormatter struct {
	logrus.JSONFormatter
}

// Format 实现自定义格式化
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// 获取调用信息
	if _, ok := entry.Data["file"]; !ok {
		// 获取调用栈信息
		if pc, file, line, ok := runtime.Caller(8); ok {
			funcName := runtime.FuncForPC(pc).Name()
			entry.Data["file"] = filepath.Base(file)
			entry.Data["line"] = line
			entry.Data["func"] = filepath.Base(funcName)
		}
	}

	// 添加时间戳格式化
	entry.Data["@timestamp"] = entry.Time.Format(time.RFC3339)

	// 添加进程信息
	entry.Data["pid"] = os.Getpid()

	// 添加协程ID
	entry.Data["goroutine_id"] = getGoroutineID()

	return f.JSONFormatter.Format(entry)
}

// Log is the global logger instance
var (
	Log  *logrus.Logger
	once sync.Once
)

// initLogger initializes the logger
func initLogger(logFilePath string) {
	Log = logrus.New()

	// 使用自定义格式化器
	Log.SetFormatter(&CustomFormatter{
		JSONFormatter: logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "@timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		},
	})

	// 创建日志目录
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		panic(fmt.Sprintf("failed to create log directory: %v", err))
	}

	// 设置日志输出
	Log.SetOutput(io.MultiWriter(os.Stdout, &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}))

	// 设置日志级别
	Log.SetLevel(logrus.DebugLevel)

	// 添加堆栈跟踪
	Log.SetReportCaller(true)
}

// GetLogger returns the singleton logger instance
func GetLogger() *logrus.Logger {
	once.Do(func() { initLogger("/Users/bianniu/GolandProjects/go_mock_server/log/app.log") })
	return Log
}

// getGoroutineID 获取当前协程ID
func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	// 解析协程ID
	var id uint64
	fmt.Sscanf(string(b), "goroutine %d", &id)
	return id
}
