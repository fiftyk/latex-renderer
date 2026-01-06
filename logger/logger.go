package logger

import "io"

// Logger 日志接口
type Logger interface {
	// Print 输出日志
	Print(v ...interface{})
	// Printf 格式化输出日志
	Printf(format string, v ...interface{})
	// Println 换行输出日志
	Println(v ...interface{})
	// Fatal 致命错误并退出
	Fatal(v ...interface{})
	// Fatalf 格式化致命错误并退出
	Fatalf(format string, v ...interface{})
	// Writer 返回日志 writer（用于 gin）
	Writer() io.Writer
}

// Level 日志级别
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Config 日志配置
type Config struct {
	Path     string // 日志文件路径，空则输出到 stdout
	MaxSize  int    // 单个日志文件最大尺寸 MB，默认 100
	MaxFiles int    // 保留的日志文件数量，默认 3
	Level    Level  // 日志级别
}
