package renderer

import (
	"context"
	"sync"
	"time"
)

// batchRenderer 批量渲染器实现
type batchRenderer struct {
	renderer    RendererInterface
	config      *BatchConfig
	queue       chan batchRequest
	requests    []batchRequest
	mu          sync.Mutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
	stats       BatchStats
	running     bool
	windowTimer *time.Timer
}

// RendererInterface 渲染器接口（用于测试）
type RendererInterface interface {
	Render(ctx context.Context, opts *RenderOptions) ([]byte, error)
}

// NewBatchRenderer 创建批量渲染器
func NewBatchRenderer(r RendererInterface, cfg *BatchConfig) BatchRendererInterface {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	if cfg.BatchWindow <= 0 {
		cfg.BatchWindow = 100 * time.Millisecond
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}

	return &batchRenderer{
		renderer: r,
		config:   cfg,
		queue:    make(chan batchRequest, cfg.QueueSize),
		requests: make([]batchRequest, 0),
		stopCh:   make(chan struct{}),
	}
}

// Enqueue 将渲染请求加入队列，返回结果 channel
func (b *batchRenderer) Enqueue(ctx context.Context, opts *RenderOptions) (<-chan *BatchResult, error) {
	resultCh := make(chan *BatchResult, 1)

	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		resultCh <- &BatchResult{Err: ErrBatchRendererStopped}
		close(resultCh)
		return resultCh, ErrBatchRendererStopped
	}
	b.mu.Unlock()

	select {
	case b.queue <- batchRequest{opts: opts, result: resultCh}:
		return resultCh, nil
	default:
		resultCh <- &BatchResult{Err: ErrQueueFull}
		close(resultCh)
		return resultCh, ErrQueueFull
	}
}

// Start 启动批处理调度器
func (b *batchRenderer) Start() {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	b.mu.Unlock()

	b.wg.Add(1)
	go b.scheduler()
}

// Stop 停止批处理调度器
func (b *batchRenderer) Stop() error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = false
	b.mu.Unlock()

	close(b.stopCh)
	b.wg.Wait()

	// 清空队列中的请求
	for {
		select {
		case req := <-b.queue:
			req.result <- &BatchResult{Err: ErrBatchRendererStopped}
		default:
			goto done
		}
	}

done:
	if b.windowTimer != nil {
		b.windowTimer.Stop()
	}

	return nil
}

// Stats 返回统计信息
func (b *batchRenderer) Stats() BatchStats {
	b.mu.Lock()
	defer b.mu.Unlock()

	return BatchStats{
		Pending:   int64(len(b.queue)),
		Processed: b.stats.Processed,
		Batches:   b.stats.Batches,
	}
}

// scheduler 调度主循环
func (b *batchRenderer) scheduler() {
	defer b.wg.Done()

	for {
		select {
		case <-b.stopCh:
			return
		case req := <-b.queue:
			b.addRequest(req)
			b.tryProcessBatch()
		}
	}
}

// addRequest 添加请求到批处理队列
func (b *batchRenderer) addRequest(req batchRequest) {
	b.requests = append(b.requests, req)
}

// tryProcessBatch 尝试处理批量任务
func (b *batchRenderer) tryProcessBatch() {
	// 检查是否满足批量大小条件
	if len(b.requests) >= b.config.BatchSize {
		b.processBatch()
		return
	}

	// 如果是第一个请求，启动时间窗口定时器
	if len(b.requests) == 1 {
		b.startWindowTimer()
	}
}

// startWindowTimer 启动时间窗口定时器
func (b *batchRenderer) startWindowTimer() {
	if b.windowTimer == nil {
		b.windowTimer = time.NewTimer(b.config.BatchWindow)
	} else {
		b.windowTimer.Reset(b.config.BatchWindow)
	}

	go func() {
		select {
		case <-b.windowTimer.C:
			b.processBatch()
		case <-b.stopCh:
			if b.windowTimer != nil {
				b.windowTimer.Stop()
			}
		}
	}()
}

// processBatch 处理批量任务
func (b *batchRenderer) processBatch() {
	b.mu.Lock()
	requests := b.requests
	b.requests = make([]batchRequest, 0)
	b.mu.Unlock()

	if len(requests) == 0 {
		return
	}

	// 批量渲染
	results := b.renderBatch(context.Background(), requests)

	// 发送结果到各请求
	for i, req := range requests {
		req.result <- results[i]
	}

	// 更新统计
	b.mu.Lock()
	b.stats.Processed += int64(len(requests))
	b.stats.Batches++
	b.mu.Unlock()
}

// renderBatch 批量渲染
func (b *batchRenderer) renderBatch(ctx context.Context, requests []batchRequest) []*BatchResult {
	results := make([]*BatchResult, len(requests))

	for i, req := range requests {
		var data []byte
		var err error
		if b.renderer != nil {
			data, err = b.renderer.Render(ctx, req.opts)
		} else {
			err = ErrBatchRendererStopped
		}
		results[i] = &BatchResult{
			Data: data,
			Err:  err,
		}
	}

	return results
}

// 错误定义
var (
	ErrBatchRendererStopped = &batchError{"batch renderer stopped"}
	ErrQueueFull            = &batchError{"queue is full"}
)

type batchError struct {
	msg string
}

func (e *batchError) Error() string {
	return e.msg
}
