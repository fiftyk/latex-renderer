package renderer

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// Renderer LaTeX 渲染器
type Renderer struct {
	browserManager   *browserManager
	renderTimeout    time.Duration // 单次渲染超时时间
	maxRetries       int           // 渲染失败最大重试次数
	overloadStrategy OverloadStrategy
	htmlGenerator    *htmlGenerator
}

// NewRenderer 创建渲染器
func NewRenderer(ropts *RendererOptions) (*Renderer, error) {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.WindowSize(1920, 1080),
	}

	// 添加额外的 Chrome 参数
	if ropts.Args != "" {
		for _, part := range strings.Fields(ropts.Args) {
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

	if ropts.ExecPath != "" {
		opts = append(opts, chromedp.ExecPath(ropts.ExecPath))
	}

	// 默认值
	maxConcurrent := ropts.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 16
	}
	maxRequests := ropts.MaxRequests
	if maxRequests <= 0 {
		maxRequests = 100
	}
	maxInterval := ropts.MaxInterval
	if maxInterval <= 0 {
		maxInterval = 30 * time.Minute
	}
	renderTimeout := ropts.RenderTimeout
	if renderTimeout <= 0 {
		renderTimeout = 30 * time.Second
	}
	maxRetries := ropts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2
	}

	// 创建浏览器管理器
	browserMgr := newBrowserManager(opts, maxRequests, maxInterval)

	// 创建HTML生成器
	cssURL := "file:///app/static/katex/katex.min.css"
	jsURL := "file:///app/static/katex/katex.min.js"
	if ropts.StaticBaseURL != "" {
		cssURL = ropts.StaticBaseURL + "/katex/katex.min.css"
		jsURL = ropts.StaticBaseURL + "/katex/katex.min.js"
	}
	htmlGen := NewHTMLGenerator(cssURL, jsURL)

	// 设置策略
	strategy := ropts.Strategy
	if strategy == nil {
		strategy = NewFailFastStrategy(maxConcurrent)
	}

	return &Renderer{
		browserManager:   browserMgr,
		renderTimeout:    renderTimeout,
		maxRetries:       maxRetries,
		overloadStrategy: strategy,
		htmlGenerator:    htmlGen,
	}, nil
}

// Render 渲染 LaTeX 为图片
func (r *Renderer) Render(ctx context.Context, opts *RenderOptions) ([]byte, error) {
	// 注意：并发限制由 Handler 层统一检查，这里不再检查

	// 检查是否需要重启浏览器
	if r.browserManager.shouldRestart() {
		if err := r.browserManager.restartBrowser(ctx); err != nil {
			return nil, fmt.Errorf("重启浏览器失败: %w", err)
		}
	}

	// 初始化浏览器（懒加载）
	if err := r.browserManager.initBrowser(ctx); err != nil {
		return nil, err
	}

	// 增加请求计数
	r.browserManager.incrementRequestCount()

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

	// 生成 HTML
	html := r.htmlGenerator.GenerateHTML(opts.Latex, background, padding, fontSize, color)

	// 使用复用的 allocator context 创建新 browser context（tab）
	browserCtx, cancel := chromedp.NewContext(r.browserManager.getAllocatorContext())
	defer cancel()

	// 设置超时
	renderCtx, cancel := context.WithTimeout(browserCtx, r.renderTimeout)
	defer cancel()

	// 执行渲染 - 添加重试机制
	var buf []byte
	var err error

	// 最多重试 maxRetries 次
	for attempt := 0; attempt < r.maxRetries; attempt++ {
		if attempt > 0 {
			// 重试前重启浏览器
			log.Println("渲染失败，重启浏览器...")
			if restartErr := r.browserManager.restartBrowser(ctx); restartErr != nil {
				return nil, fmt.Errorf("重启浏览器失败: %w", restartErr)
			}
			// 重新初始化
			if initErr := r.browserManager.initBrowser(ctx); initErr != nil {
				return nil, initErr
			}
			// 创建新的browser context
			browserCtx, cancel = chromedp.NewContext(r.browserManager.getAllocatorContext())
			defer cancel()
			renderCtx, cancel = context.WithTimeout(browserCtx, r.renderTimeout)
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
		if attempt == r.maxRetries-1 {
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

// Warmup 预热 Chrome 浏览器
func (r *Renderer) Warmup(ctx context.Context) error {
	return r.browserManager.Warmup(ctx)
}

// Close 关闭渲染器
func (r *Renderer) Close() error {
	return r.browserManager.Close()
}
