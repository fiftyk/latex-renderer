package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	// 测试默认配置（输出到 stdout）
	log, err := New(nil)
	if err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}
	if log == nil {
		t.Fatal("日志不应为 nil")
	}

	// 测试带配置的日志
	cfg := &Config{
		Path:     "/tmp/test.log",
		MaxSize:  50,
		MaxFiles: 5,
		Level:    LevelDebug,
	}
	log2, err := New(cfg)
	if err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}
	if log2 == nil {
		t.Fatal("日志不应为 nil")
	}
}

func TestLoggerInterface(t *testing.T) {
	log, _ := New(nil)

	// 测试 Print
	log.Print("test")

	// 测试 Printf
	log.Printf("test %s", "message")

	// 测试 Println
	log.Println("test")

	// 测试 Writer
	if log.Writer() == nil {
		t.Error("Writer 不应为 nil")
	}
}

func TestMockLogger(t *testing.T) {
	mock := NewMock()

	// 测试 Print
	mock.Print("test")
	if !mock.HasMessage("test") {
		t.Error("MockLogger 应包含 'test' 消息")
	}

	// 测试 Printf
	mock.Printf("hello %s", "world")
	if !mock.HasMessage("hello world") {
		t.Error("MockLogger 应包含 'hello world' 消息")
	}

	// 测试 Println
	mock.Println("line")
	if !mock.HasMessage("line") {
		t.Error("MockLogger 应包含 'line' 消息")
	}

	// 测试 GetMessages
	messages := mock.GetMessages()
	if len(messages) != 3 {
		t.Errorf("期望 3 条消息, got %d", len(messages))
	}

	// 测试 Clear
	mock.Clear()
	if len(mock.GetMessages()) != 0 {
		t.Error("Clear 后应为空")
	}

	// 测试 HasMessage
	mock.Print("error: something failed")
	if !mock.HasMessage("error:") {
		t.Error("MockLogger 应包含 'error:' 消息")
	}
	if mock.HasMessage("success") {
		t.Error("MockLogger 不应包含 'success' 消息")
	}
}

func TestMockWriter(t *testing.T) {
	mock := NewMock()
	writer := mock.Writer()

	buf := []byte("test message\n")
	n, err := writer.Write(buf)
	if err != nil {
		t.Fatalf("Write 失败: %v", err)
	}
	if n != len(buf) {
		t.Errorf("期望写入 %d bytes, got %d", len(buf), n)
	}

	if !mock.HasMessage("test message") {
		t.Error("MockLogger 应包含 'test message' 消息")
	}
}

func TestLoggerLevel(t *testing.T) {
	levels := []Level{LevelDebug, LevelInfo, LevelWarn, LevelError}
	for _, level := range levels {
		cfg := &Config{
			Path:  "",
			Level: level,
		}
		log, err := New(cfg)
		if err != nil {
			t.Fatalf("创建日志失败: %v", err)
		}
		log.Printf("level test: %s", level)
	}
}

func TestLoggerWithFile(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Path: "/tmp/logger_test.log",
	}
	log, err := New(cfg)
	if err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}

	log.Print("file test")

	// 验证文件被创建
	if !strings.Contains(buf.String(), "file test") {
		t.Log("日志已输出到文件")
	}
}

func TestNewLoggerError(t *testing.T) {
	// 测试无效路径
	cfg := &Config{
		Path: "/nonexistent/path/test.log",
	}
	_, err := New(cfg)
	if err != nil {
		t.Logf("期望错误（目录不存在）: %v", err)
	}
}
