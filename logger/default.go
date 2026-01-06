package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// defaultLogger 默认日志实现
type defaultLogger struct {
	logger *log.Logger
	writer io.Writer
}

// New 创建日志实例
func New(cfg *Config) (Logger, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	// 默认值
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 100
	}
	if cfg.MaxFiles == 0 {
		cfg.MaxFiles = 3
	}
	if cfg.Level == "" {
		cfg.Level = LevelInfo
	}

	// 默认输出到 stdout
	if cfg.Path == "" {
		l := log.New(os.Stdout, "", log.LstdFlags)
		return &defaultLogger{logger: l, writer: os.Stdout}, nil
	}

	// 确保日志目录存在
	logDir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 打开日志文件（追加模式）
	file, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 同时输出到 stdout（方便 docker logs 查看）
	multiWriter := io.MultiWriter(file, os.Stdout)
	l := log.New(multiWriter, "", log.LstdFlags)

	return &defaultLogger{logger: l, writer: multiWriter}, nil
}

func (l *defaultLogger) Print(v ...interface{}) {
	l.logger.Print(v...)
}

func (l *defaultLogger) Printf(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}

func (l *defaultLogger) Println(v ...interface{}) {
	l.logger.Println(v...)
}

func (l *defaultLogger) Fatal(v ...interface{}) {
	l.logger.Fatal(v...)
}

func (l *defaultLogger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatalf(format, v...)
}

func (l *defaultLogger) Writer() io.Writer {
	return l.writer
}
