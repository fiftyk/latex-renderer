package renderer

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

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

// Renderer LaTeX 渲染器 (复用 Chrome browser 实例)
type Renderer struct {
	allocOpts        []chromedp.ExecAllocatorOption
	allocCtx         context.Context
	allocCancel      context.CancelFunc
	browserMu        sync.RWMutex
	initialized      bool
	requestCount     int64         // 请求计数，用于定期重启
	maxRequests      int64         // 最大请求数后重启浏览器
	lastRestart      time.Time     // 上次重启时间
	maxInterval      time.Duration // 最大间隔时间
	overloadStrategy OverloadStrategy
	staticBaseURL    string // 静态资源基础URL
}

// RenderOptions 渲染选项
type RenderOptions struct {
	Latex      string // LaTeX 公式
	Color      string // 字体颜色 (默认 black)
	Background string // 背景颜色 (默认 transparent)
	FontSize   string // 字体大小 px (默认 16)
	Padding    string // 内边距 px (默认 20)
}

// NewRenderer 创建渲染器
func NewRenderer(execPath, args string, maxConcurrent int, staticBaseURL ...string) (*Renderer, error) {
	var baseURL string
	if len(staticBaseURL) > 0 {
		baseURL = staticBaseURL[0]
	}
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.WindowSize(1920, 1080),
	}

	// 添加额外的 Chrome 参数
	if args != "" {
		for _, part := range strings.Fields(args) {
			if strings.HasPrefix(part, "--") {
				// 处理 --flag=value 格式
				if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
					opts = append(opts, chromedp.Flag(kv[0][2:], kv[1]))
				} else {
					// 处理 --flag 格式（无值参数如 --no-sandbox）
					opts = append(opts, chromedp.Flag(part[2:], "true"))
				}
			}
		}
	}

	if execPath != "" {
		opts = append(opts, chromedp.ExecPath(execPath))
	}

	// 默认并发数为 2
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}

	return &Renderer{
		allocOpts:        opts,
		maxRequests:      10,               // 每10个请求重启一次（更频繁重启避免崩溃）
		maxInterval:      5 * time.Minute,  // 或每5分钟重启一次（更频繁重启避免崩溃）
		lastRestart:      time.Now(),
		overloadStrategy: NewFailFastStrategy(maxConcurrent),
		staticBaseURL:    baseURL,
	}, nil
}

// Warmup 预热 Chrome 浏览器，初始化一次后可复用
func (r *Renderer) Warmup(ctx context.Context) error {
	r.browserMu.Lock()
	defer r.browserMu.Unlock()

	if r.initialized {
		return nil
	}

	// 创建 allocator context（长期存活）
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, r.allocOpts...)

	r.allocCtx = allocCtx
	r.allocCancel = cancel
	r.initialized = true

	return nil
}

// initBrowser 初始化浏览器（懒加载）
func (r *Renderer) initBrowser(ctx context.Context) error {
	r.browserMu.RLock()
	if r.initialized {
		r.browserMu.RUnlock()
		return nil
	}
	r.browserMu.RUnlock()

	r.browserMu.Lock()
	defer r.browserMu.Unlock()

	// 双重检查
	if r.initialized {
		return nil
	}

	// 创建 allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, r.allocOpts...)

	r.allocCtx = allocCtx
	r.allocCancel = cancel
	r.initialized = true

	return nil
}

// shouldRestart 检查是否需要重启浏览器
func (r *Renderer) shouldRestart() bool {
	r.browserMu.Lock()
	defer r.browserMu.Unlock()

	if !r.initialized {
		return false
	}

	// 检查请求数
	if r.requestCount >= r.maxRequests {
		return true
	}

	// 检查时间间隔
	if time.Since(r.lastRestart) >= r.maxInterval {
		return true
	}

	return false
}

// restartBrowser 重启浏览器
func (r *Renderer) restartBrowser(ctx context.Context) error {
	r.browserMu.Lock()
	defer r.browserMu.Unlock()

	// 取消旧的allocator context
	if r.allocCancel != nil {
		r.allocCancel()
		r.allocCancel = nil
	}

	// 短暂延迟确保旧实例被清理
	time.Sleep(100 * time.Millisecond)

	// 创建新的 allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, r.allocOpts...)
	r.allocCtx = allocCtx
	r.allocCancel = cancel
	r.requestCount = 0
	r.lastRestart = time.Now()

	// 再次短暂延迟确保新实例启动
	time.Sleep(200 * time.Millisecond)

	return nil
}

// Render 渲染 LaTeX 为图片 (复用 browser 实例)
func (r *Renderer) Render(ctx context.Context, opts *RenderOptions) ([]byte, error) {
	// 注意：并发限制由 Handler 层统一检查，这里不再检查

	// 检查是否需要重启浏览器
	if r.shouldRestart() {
		if err := r.restartBrowser(ctx); err != nil {
			return nil, fmt.Errorf("重启浏览器失败: %w", err)
		}
	}

	// 初始化浏览器（懒加载）
	if err := r.initBrowser(ctx); err != nil {
		return nil, err
	}

	// 增加请求计数
	r.browserMu.Lock()
	r.requestCount++
	r.browserMu.Unlock()

	// 设置默认值
	color := opts.Color
	if color == "" {
		color = "black"
	}
	background := opts.Background
	if background == "" {
		background = "transparent"
	}
	fontSize := opts.FontSize
	if fontSize == "" {
		fontSize = "16"
	}
	padding := opts.Padding
	if padding == "" {
		padding = "20"
	}

	// 生成 HTML（使用HTTP服务器提供KaTeX资源）
	cssURL := "file:///app/static/katex/katex.min.css"
	jsURL := "file:///app/static/katex/katex.min.js"
	if r.staticBaseURL != "" {
		cssURL = r.staticBaseURL + "/katex/katex.min.css"
		jsURL = r.staticBaseURL + "/katex/katex.min.js"
	}
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="%s">
  <script src="%s"></script>
  <style>
    body {
      margin: 0;
      padding: 0;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      background-color: %s;
    }
    .katex-wrapper {
      padding: %spx;
    }
    .katex {
      font-size: %spx;
      color: %s;
    }
    .katex-display {
      margin: 0;
    }
  </style>
</head>
<body>
  <div class="katex-wrapper" id="wrapper">
    <div id="formula"></div>
  </div>
  <script>
    katex.render(%q, document.getElementById('formula'), {
      displayMode: true,
      throwOnError: false,
      output: 'html'
    });
  </script>
</body>
</html>`, cssURL, jsURL, background, padding, fontSize, color, opts.Latex)

	// 使用复用的 allocator context 创建新 browser context（tab）
	browserCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	// 设置超时 - 缩短到10秒，快速失败
	renderCtx, cancel := context.WithTimeout(browserCtx, 10*time.Second)
	defer cancel()

	// 执行渲染 - 添加重试机制
	var buf []byte
	var err error

	// 最多重试2次
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			// 重试前重启浏览器
			log.Println("渲染失败，重启浏览器...")
			if restartErr := r.restartBrowser(ctx); restartErr != nil {
				return nil, fmt.Errorf("重启浏览器失败: %w", restartErr)
			}
			// 重新初始化
			if initErr := r.initBrowser(ctx); initErr != nil {
				return nil, initErr
			}
			// 创建新的browser context
			browserCtx, cancel = chromedp.NewContext(r.allocCtx)
			defer cancel()
			renderCtx, cancel = context.WithTimeout(browserCtx, 10*time.Second)
			defer cancel()
		}

		err = chromedp.Run(renderCtx,
			emulation.SetDeviceMetricsOverride(1920, 1080, 1.0, false),
			chromedp.Navigate(`data:text/html,`+html),
			chromedp.WaitVisible(`#wrapper`, chromedp.ByQuery),
			chromedp.WaitVisible(`.katex`, chromedp.ByQuery),
			chromedp.Screenshot(`#wrapper`, &buf, chromedp.ByQuery),
		)

		if err == nil {
			// 成功！
			break
		}

		// 如果是最后一次尝试，返回错误
		if attempt == 1 {
			return nil, fmt.Errorf("渲染失败 (尝试 %d 次): %w", attempt+1, err)
		}

		// 短暂延迟后重试
		time.Sleep(500 * time.Millisecond)
	}

	return buf, nil
}

// RenderToPNG 渲染为 PNG
func (r *Renderer) RenderToPNG(ctx context.Context, opts *RenderOptions) ([]byte, error) {
	return r.Render(ctx, opts)
}

// Close 关闭渲染器
func (r *Renderer) Close() error {
	r.browserMu.Lock()
	defer r.browserMu.Unlock()

	if r.allocCancel != nil {
		r.allocCancel()
		r.allocCancel = nil
	}
	r.allocCtx = nil
	r.initialized = false

	return nil
}

// FindChrome 查找 Chrome 可执行文件
func FindChrome() string {
	paths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
