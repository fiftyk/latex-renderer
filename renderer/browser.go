package renderer

import (
	"context"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// browserManager 浏览器生命周期管理器
type browserManager struct {
	allocOpts    []chromedp.ExecAllocatorOption
	allocCtx     context.Context
	allocCancel  context.CancelFunc
	browserMu    sync.RWMutex
	initialized  bool
	requestCount int64         // 请求计数，用于定期重启
	maxRequests  int64         // 最大请求数后重启浏览器
	lastRestart  time.Time     // 上次重启时间
	maxInterval  time.Duration // 最大间隔时间
}

// newBrowserManager 创建新的浏览器管理器
func newBrowserManager(
	opts []chromedp.ExecAllocatorOption,
	maxRequests int64,
	maxInterval time.Duration,
) *browserManager {
	return &browserManager{
		allocOpts:   opts,
		maxRequests: maxRequests,
		maxInterval: maxInterval,
		lastRestart: time.Now(),
	}
}

// Warmup 预热 Chrome 浏览器，初始化一次后可复用
func (bm *browserManager) Warmup(ctx context.Context) error {
	bm.browserMu.Lock()
	defer bm.browserMu.Unlock()

	if bm.initialized {
		return nil
	}

	// 创建 allocator context（长期存活）
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, bm.allocOpts...)

	bm.allocCtx = allocCtx
	bm.allocCancel = cancel
	bm.initialized = true

	return nil
}

// initBrowser 初始化浏览器（懒加载）
func (bm *browserManager) initBrowser(ctx context.Context) error {
	bm.browserMu.RLock()
	if bm.initialized {
		bm.browserMu.RUnlock()
		return nil
	}
	bm.browserMu.RUnlock()

	bm.browserMu.Lock()
	defer bm.browserMu.Unlock()

	// 双重检查
	if bm.initialized {
		return nil
	}

	// 创建 allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, bm.allocOpts...)

	bm.allocCtx = allocCtx
	bm.allocCancel = cancel
	bm.initialized = true

	return nil
}

// shouldRestart 检查是否需要重启浏览器
func (bm *browserManager) shouldRestart() bool {
	bm.browserMu.Lock()
	defer bm.browserMu.Unlock()

	if !bm.initialized {
		return false
	}

	// 检查请求数
	if bm.requestCount >= bm.maxRequests {
		return true
	}

	// 检查时间间隔
	if time.Since(bm.lastRestart) >= bm.maxInterval {
		return true
	}

	return false
}

// restartBrowser 重启浏览器
func (bm *browserManager) restartBrowser(ctx context.Context) error {
	bm.browserMu.Lock()
	defer bm.browserMu.Unlock()

	// 取消旧的allocator context
	if bm.allocCancel != nil {
		bm.allocCancel()
		bm.allocCancel = nil
	}

	// 短暂延迟确保旧实例被清理
	time.Sleep(100 * time.Millisecond)

	// 创建新的 allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, bm.allocOpts...)
	bm.allocCtx = allocCtx
	bm.allocCancel = cancel
	bm.requestCount = 0
	bm.lastRestart = time.Now()

	// 再次短暂延迟确保新实例启动
	time.Sleep(200 * time.Millisecond)

	return nil
}

// incrementRequestCount 增加请求计数
func (bm *browserManager) incrementRequestCount() {
	bm.browserMu.Lock()
	defer bm.browserMu.Unlock()

	bm.requestCount++
}

// getAllocatorContext 获取allocator context
func (bm *browserManager) getAllocatorContext() context.Context {
	bm.browserMu.RLock()
	defer bm.browserMu.RUnlock()
	return bm.allocCtx
}

// isInitialized 检查是否已初始化
func (bm *browserManager) isInitialized() bool {
	bm.browserMu.RLock()
	defer bm.browserMu.RUnlock()
	return bm.initialized
}

// Close 关闭浏览器管理器
func (bm *browserManager) Close() error {
	bm.browserMu.Lock()
	defer bm.browserMu.Unlock()

	if bm.allocCancel != nil {
		bm.allocCancel()
		bm.allocCancel = nil
	}
	bm.allocCtx = nil
	bm.initialized = false

	return nil
}
