package renderer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/emulation"
)

// Renderer LaTeX 渲染器 (复用 Chrome browser 实例)
type Renderer struct {
	allocOpts   []chromedp.ExecAllocatorOption
	allocCtx    context.Context
	allocCancel context.CancelFunc
	browserMu   sync.RWMutex
	initialized bool
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
func NewRenderer(execPath, args string) (*Renderer, error) {
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
				if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
					opts = append(opts, chromedp.Flag(kv[0][2:], kv[1]))
				}
			}
		}
	}

	if execPath != "" {
		opts = append(opts, chromedp.ExecPath(execPath))
	}

	return &Renderer{
		allocOpts: opts,
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

// Render 渲染 LaTeX 为图片 (复用 browser 实例)
func (r *Renderer) Render(ctx context.Context, opts *RenderOptions) ([]byte, error) {
	// 初始化浏览器（懒加载）
	if err := r.initBrowser(ctx); err != nil {
		return nil, err
	}

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

	// 生成 HTML（padding 应用到 wrapper 容器，截图整个 wrapper）
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.css">
  <script src="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.js"></script>
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
</html>`, background, padding, fontSize, color, opts.Latex)

	// 使用复用的 allocator context 创建新 browser context（tab）
	browserCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	// 设置超时
	ctx, cancel = context.WithTimeout(browserCtx, 60*time.Second)
	defer cancel()

	// 执行渲染
	var buf []byte

	err := chromedp.Run(ctx,
		emulation.SetDeviceMetricsOverride(1920, 1080, 1.0, false),
		chromedp.Navigate(`data:text/html,`+html),
		chromedp.WaitVisible(`#wrapper`, chromedp.ByQuery),
		chromedp.Screenshot(`#wrapper`, &buf, chromedp.ByQuery),
	)

	if err != nil {
		return nil, fmt.Errorf("渲染失败: %w", err)
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
