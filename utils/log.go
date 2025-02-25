package utils

import (
	"io"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Log is the global logger instance
var (
	Log  *logrus.Logger
	once sync.Once
)

// initLogger initializes the logger
func initLogger(logFilePath string) {
	Log = logrus.New()

	// 设置日志格式为JSON格式
	Log.SetFormatter(&logrus.JSONFormatter{})

	// 设置日志输出到文件和控制台
	Log.SetOutput(io.MultiWriter(os.Stdout, &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10,   // 每个日志文件的最大大小（MB）
		MaxBackups: 5,    // 保留的旧日志文件的最大数量
		MaxAge:     30,   // 保留的旧日志文件的最大天数
		Compress:   true, // 是否压缩旧的日志文件
	}))

	// 设置日志级别
	Log.SetLevel(logrus.DebugLevel)
}

// GetLogger returns the singleton logger instance
func GetLogger() *logrus.Logger {
	once.Do(func() { initLogger("/Users/bianniu/GolandProjects/go_mock_server/log/app.log") })
	return Log
}
