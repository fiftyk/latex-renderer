package renderer

import (
	"testing"
	"time"
)

func TestFailFastStrategy(t *testing.T) {
	limit := 2
	strategy := NewFailFastStrategy(limit)

	// 第一次获取应该成功
	if !strategy.Handle() {
		t.Error("First Handle() should succeed")
	}
	defer strategy.Release()

	// 第二次获取应该成功
	if !strategy.Handle() {
		t.Error("Second Handle() should succeed")
	}
	defer strategy.Release()

	// 第三次获取应该失败（达到限制）
	if strategy.Handle() {
		t.Error("Third Handle() should fail when limit is reached")
	}
}

func TestTimeoutQueueStrategy_HandleTimeout(t *testing.T) {
	limit := 1
	queueSize := 1  // 额外队列大小 1，总容量 = 1 + 1 = 2
	timeout := 100 * time.Millisecond
	strategy := NewTimeoutQueueStrategy(limit, queueSize, timeout)

	// 占用所有槽位 (limit + queueSize = 2)
	if !strategy.Handle() {
		t.Error("First Handle() should succeed")
	}
	if !strategy.Handle() {
		t.Error("Second Handle() should succeed")
	}

	// 第三次获取应该超时失败
	start := time.Now()
	if strategy.Handle() {
		t.Error("Third Handle() should fail due to timeout")
	}
	elapsed := time.Since(start)

	// 应该在大约 timeout 时间内返回
	if elapsed < timeout/2 || elapsed > timeout*2 {
		t.Errorf("Timeout should be around %v, got %v", timeout, elapsed)
	}

	// 释放一个槽位
	strategy.Release()

	// 现在应该可以再次获取
	if !strategy.Handle() {
		t.Error("Handle() should succeed after Release()")
	}
	strategy.Release()
}

func TestTimeoutQueueStrategy_NoMemoryLeak(t *testing.T) {
	limit := 1
	queueSize := 1
	timeout := 50 * time.Millisecond
	strategy := NewTimeoutQueueStrategy(limit, queueSize, timeout)

	// 快速触发多次超时，检查是否有内存泄漏
	for i := 0; i < 10; i++ {
		strategy.Handle() // 这个应该超时
	}

	// 如果有 time.After 导致的 timer 泄漏，heap 可能会增长
	// 这里只是基本验证不 panic，实际内存测试需要更复杂的工具
}

func TestTimeoutQueueStrategy_RejectMessage(t *testing.T) {
	limit := 1
	queueSize := 1
	timeout := 100 * time.Millisecond
	strategy := NewTimeoutQueueStrategy(limit, queueSize, timeout)

	// 占用槽位
	strategy.Handle()

	// 触发拒绝
	strategy.Handle()

	err := strategy.Reject()
	if err == nil {
		t.Error("Reject() should return an error")
	}
}
