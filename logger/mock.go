package logger

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// MockLogger 用于测试的模拟日志
type MockLogger struct {
	mu      sync.Mutex
	messages []string
}

// NewMock 创建模拟日志
func NewMock() *MockLogger {
	return &MockLogger{
		messages: make([]string, 0),
	}
}

func (l *MockLogger) Print(v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprint(v...))
}

func (l *MockLogger) Printf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintf(format, v...))
}

func (l *MockLogger) Println(v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintln(v...))
}

func (l *MockLogger) Fatal(v ...interface{}) {
	l.Print(v...)
	panic("fatal")
}

func (l *MockLogger) Fatalf(format string, v ...interface{}) {
	l.Printf(format, v...)
	panic("fatal")
}

func (l *MockLogger) Writer() io.Writer {
	return &mockWriter{logger: l}
}

// GetMessages 获取所有日志消息
func (l *MockLogger) GetMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string{}, l.messages...)
}

// HasMessage 检查是否包含指定消息
func (l *MockLogger) HasMessage(substring string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, msg := range l.messages {
		if strings.Contains(msg, substring) {
			return true
		}
	}
	return false
}

// Clear 清空消息
func (l *MockLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = l.messages[:0]
}

type mockWriter struct {
	logger *MockLogger
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	w.logger.Print(string(p))
	return len(p), nil
}
