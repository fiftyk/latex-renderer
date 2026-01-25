package renderer

import (
	"fmt"
	"time"
)

// TimeoutQueueConfig TimeoutQueueStrategy 配置（用于测试）
type TimeoutQueueConfig struct {
	Limit     int
	QueueSize int
	Timeout   time.Duration
}

// OverloadStrategy 并发满时的处理策略接口
type OverloadStrategy interface {
	// Handle 尝试获取信号量，返回 true 表示获取成功
	Handle() bool
	// Release 释放信号量
	Release()
	// Reject 返回被拒绝时的错误
	Reject() error
}

// FailFastStrategy 快速失败策略：并发满时立即返回错误
type FailFastStrategy struct {
	sem chan struct{}
}

// NewFailFastStrategy 创建快速失败策略
func NewFailFastStrategy(limit int) OverloadStrategy {
	return &FailFastStrategy{
		sem: make(chan struct{}, limit),
	}
}

func (s *FailFastStrategy) Handle() bool {
	select {
	case s.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *FailFastStrategy) Reject() error {
	return fmt.Errorf("服务繁忙，请稍后再试")
}

func (s *FailFastStrategy) Release() {
	select {
	case <-s.sem:
	default:
	}
}

// TimeoutQueueStrategy 超时排队策略：并发满时排队等待，超时后返回错误
type TimeoutQueueStrategy struct {
	sem       chan struct{}
	queueSize int
	timeout   time.Duration
}

// NewTimeoutQueueStrategy 创建超时排队策略
// queueSize: 队列大小（最大排队请求数）
// timeout: 排队超时时间
func NewTimeoutQueueStrategy(limit int, queueSize int, timeout time.Duration) OverloadStrategy {
	if queueSize <= 0 {
		queueSize = limit * 2 // 默认队列大小为并发数的2倍
	}
	if timeout <= 0 {
		timeout = 5 * time.Second // 默认5秒超时
	}

	return &TimeoutQueueStrategy{
		sem:       make(chan struct{}, limit+queueSize), // 总大小 = 并发 + 队列
		queueSize: queueSize,
		timeout:   timeout,
	}
}

func (s *TimeoutQueueStrategy) Handle() bool {
	// 使用 timer 避免 time.After 导致的内存泄漏
	timer := time.NewTimer(s.timeout)
	defer timer.Stop()

	// 尝试获取信号量，支持排队
	select {
	case s.sem <- struct{}{}:
		return true
	case <-timer.C:
		// 超时
		return false
	}
}

func (s *TimeoutQueueStrategy) Reject() error {
	return fmt.Errorf("服务繁忙，排队超时（%v），请稍后重试", s.timeout)
}

func (s *TimeoutQueueStrategy) Release() {
	select {
	case <-s.sem:
	default:
	}
}
