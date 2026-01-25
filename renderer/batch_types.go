package renderer

import (
	"context"
	"time"
)

// BatchRendererInterface 批量渲染器接口
type BatchRendererInterface interface {
	// Enqueue 将渲染请求加入队列，返回结果 channel
	Enqueue(ctx context.Context, opts *RenderOptions) (<-chan *BatchResult, error)

	// Start 启动批处理调度器
	Start()

	// Stop 停止批处理调度器
	Stop() error

	// Stats 返回统计信息
	Stats() BatchStats
}

// BatchResult 渲染结果
type BatchResult struct {
	Data []byte // 渲染图片数据
	Err  error  // 错误信息
}

// BatchConfig 批处理配置
type BatchConfig struct {
	BatchSize   int           // 批量大小阈值（如 10）
	BatchWindow time.Duration // 时间窗口（如 100ms）
	QueueSize   int           // 队列缓冲大小（如 100）
}

// BatchStats 统计信息
type BatchStats struct {
	Pending   int64 // 等待处理的数量
	Processed int64 // 已处理的数量
	Batches   int64 // 批次数
}

// batchRequest 内部请求结构
type batchRequest struct {
	opts   *RenderOptions
	result chan<- *BatchResult
}
