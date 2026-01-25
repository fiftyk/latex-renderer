package renderer

import (
	"context"
	"testing"
	"time"
)

// mockRenderer 用于测试的模拟渲染器
type mockRenderer struct {
	results map[string][]byte
	err     error
}

func newMockRenderer(results map[string][]byte, err error) *mockRenderer {
	return &mockRenderer{results: results, err: err}
}

func (m *mockRenderer) Render(ctx context.Context, opts *RenderOptions) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results[opts.Latex], nil
}

// TestBatchRendererInterface tests the batch renderer interface
func TestBatchRendererInterface(t *testing.T) {
	// 测试接口是否可以被正确实现
	var _ BatchRendererInterface = NewBatchRenderer(nil, &BatchConfig{})
}

// TestBatchRendererEnqueueAndProcess tests enqueuing requests and processing batches
func TestBatchRendererEnqueueAndProcess(t *testing.T) {
	config := &BatchConfig{
		BatchSize:   3,
		BatchWindow: 50 * time.Millisecond,
		QueueSize:   100,
	}

	// 创建 mock 渲染器
	mock := newMockRenderer(map[string][]byte{
		"a": []byte("data_a"),
		"b": []byte("data_b"),
		"c": []byte("data_c"),
	}, nil)

	br := NewBatchRenderer(mock, config)
	br.Start()
	defer br.Stop()

	ctx := context.Background()

	// 并发入队 3 个请求
	resultChans := make([]<-chan *BatchResult, 3)
	for i, latex := range []string{"a", "b", "c"} {
		ch, err := br.Enqueue(ctx, &RenderOptions{Latex: latex})
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
		resultChans[i] = ch
	}

	// 收集结果
	results := make(map[string]*BatchResult)
	for i, ch := range resultChans {
		result := <-ch
		results[string(rune('a'+i))] = result
	}

	// 验证结果
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	for i := 0; i < 3; i++ {
		key := string(rune('a' + i))
		result, ok := results[key]
		if !ok {
			t.Errorf("Missing result for %s", key)
			continue
		}
		if result.Err != nil {
			t.Errorf("Result for %s has error: %v", key, result.Err)
		}
		if string(result.Data) != "data_"+key {
			t.Errorf("Result for %s: expected 'data_%s', got '%s'", key, key, string(result.Data))
		}
	}
}

// TestBatchRendererSizeTrigger tests batch processing when size threshold is reached
func TestBatchRendererSizeTrigger(t *testing.T) {
	config := &BatchConfig{
		BatchSize:   2, // 2 个请求触发批量处理
		BatchWindow: 100 * time.Millisecond,
		QueueSize:   100,
	}

	mock := newMockRenderer(map[string][]byte{
		"x": []byte("result_x"),
		"y": []byte("result_y"),
	}, nil)

	br := NewBatchRenderer(mock, config)
	br.Start()
	defer br.Stop()

	ctx := context.Background()

	// 入队 2 个请求，应该立即触发批量处理
	ch1, _ := br.Enqueue(ctx, &RenderOptions{Latex: "x"})
	ch2, _ := br.Enqueue(ctx, &RenderOptions{Latex: "y"})

	// 两个结果应该都能收到
	result1 := <-ch1
	result2 := <-ch2

	if result1.Err != nil || result2.Err != nil {
		t.Error("Expected no errors")
	}
}

// TestBatchRendererWindowTrigger tests batch processing when time window is reached
func TestBatchRendererWindowTrigger(t *testing.T) {
	config := &BatchConfig{
		BatchSize:   10, // 较大的阈值，让时间窗口先触发
		BatchWindow: 30 * time.Millisecond,
		QueueSize:   100,
	}

	mock := newMockRenderer(map[string][]byte{
		"p": []byte("result_p"),
	}, nil)

	br := NewBatchRenderer(mock, config)
	br.Start()
	defer br.Stop()

	ctx := context.Background()

	// 只入队 1 个请求
	ch, _ := br.Enqueue(ctx, &RenderOptions{Latex: "p"})

	// 应该能在时间窗口内收到结果
	select {
	case result := <-ch:
		if result.Err != nil {
			t.Errorf("Unexpected error: %v", result.Err)
		}
		if string(result.Data) != "result_p" {
			t.Errorf("Expected 'result_p', got '%s'", string(result.Data))
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for result")
	}
}

// TestBatchRendererStop tests stopping the batch renderer
func TestBatchRendererStop(t *testing.T) {
	config := &BatchConfig{
		BatchSize:   10,
		BatchWindow: time.Hour, // 长时间窗口
		QueueSize:   100,
	}

	mock := newMockRenderer(map[string][]byte{}, nil)

	br := NewBatchRenderer(mock, config)
	br.Start()

	// 停止
	if err := br.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// 停止后应该不能再入队
	ctx := context.Background()
	_, err := br.Enqueue(ctx, &RenderOptions{Latex: "test"})
	if err == nil {
		t.Error("Expected error after stop")
	}
}

// TestBatchRendererStats tests statistics reporting
func TestBatchRendererStats(t *testing.T) {
	config := &BatchConfig{
		BatchSize:   2,
		BatchWindow: 50 * time.Millisecond,
		QueueSize:   100,
	}

	mock := newMockRenderer(map[string][]byte{
		"s1": []byte("data1"),
		"s2": []byte("data2"),
	}, nil)

	br := NewBatchRenderer(mock, config)
	br.Start()
	defer br.Stop()

	ctx := context.Background()

	// 入队 2 个请求
	ch1, _ := br.Enqueue(ctx, &RenderOptions{Latex: "s1"})
	ch2, _ := br.Enqueue(ctx, &RenderOptions{Latex: "s2"})

	// 等待结果
	<-ch1
	<-ch2

	// 检查统计
	stats := br.Stats()
	if stats.Processed < 2 {
		t.Errorf("Expected at least 2 processed, got %d", stats.Processed)
	}
	if stats.Batches < 1 {
		t.Errorf("Expected at least 1 batch, got %d", stats.Batches)
	}
}

// TestBatchRendererErrorHandling tests error handling in batch processing
func TestBatchRendererErrorHandling(t *testing.T) {
	config := &BatchConfig{
		BatchSize:   2,
		BatchWindow: 50 * time.Millisecond,
		QueueSize:   100,
	}

	// 创建会返回错误的 mock 渲染器
	mock := newMockRenderer(nil, assertAnError{})

	br := NewBatchRenderer(mock, config)
	br.Start()
	defer br.Stop()

	ctx := context.Background()

	// 入队请求，应该收到错误
	ch, _ := br.Enqueue(ctx, &RenderOptions{Latex: "error_test"})

	select {
	case result := <-ch:
		if result.Err == nil {
			t.Error("Expected error, got nil")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for error result")
	}
}

// assertAnError 用于测试的错误类型
type assertAnError struct{}

func (e assertAnError) Error() string {
	return "asserted error"
}
