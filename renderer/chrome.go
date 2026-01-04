package renderer

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/emulation"
)

// Renderer LaTeX 渲染器 (优化版：单浏览器 + 复用 tab)
type Renderer struct {
	browser    context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	template   *template.Template
}

// NewRenderer 创建渲染器 (启动驻留浏览器)
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

	// 创建驻留浏览器
	browserCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(browserCtx, chromedp.WithLogf(log.Printf))

	// 启动浏览器
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("启动浏览器失败: %w", err)
	}

	// 解析 HTML 模板
	tmpl := `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.css">
  <script src="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.js"></script>
  <style>
    body {
      margin: 0;
      padding: 20px;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      background-color: transparent;
    }
    .katex {
      font-size: {{.Scale}}em;
      color: {{.Color}};
    }
  </style>
</head>
<body>
  <div id="formula"></div>
  <script>
    katex.render("{{.Latex}}", document.getElementById('formula'), {
      displayMode: true,
      throwOnError: false,
      output: 'html'
    });
  </script>
</body>
</html>`

	t, err := template.New("latex").Parse(tmpl)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("解析模板失败: %w", err)
	}

	return &Renderer{
		browser:  ctx,
		cancel:   cancel,
		template: t,
	}, nil
}

// RenderRequest 渲染请求
type RenderRequest struct {
	Latex  string
	Scale  float64
	Color  string
}

// RenderResult 渲染结果
type RenderResult struct {
	Data   []byte
	Width  int
	Height int
}

// Render 渲染 LaTeX 为图片 (复用 tab，无需重新启动浏览器)
func (r *Renderer) Render(ctx context.Context, req *RenderRequest) (*RenderResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 生成 HTML
	html := r.generateHTML(req)

	// 设置超时
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 执行渲染 - 复用同一个 tab
	var buf []byte

	err := chromedp.Run(timeoutCtx,
		// 设置视口
		emulation.SetDeviceMetricsOverride(1920, 1080, 1.0, false),
		// 导航到 HTML
		chromedp.Navigate(`data:text/html,`+html),
		// 等待 KaTeX 渲染完成
		chromedp.WaitVisible(`.katex`, chromedp.ByQuery),
		// 截图
		chromedp.Screenshot(`.katex`, &buf, chromedp.ByQuery),
	)

	if err != nil {
		return nil, fmt.Errorf("渲染失败: %w", err)
	}

	return &RenderResult{
		Data:   buf,
		Width:  1920,
		Height: 1080,
	}, nil
}

// RenderToPNG 渲染为 PNG
func (r *Renderer) RenderToPNG(ctx context.Context, latex string, scale, color string) ([]byte, error) {
	scaleFloat := 2.0
	fmt.Sscanf(scale, "%f", &scaleFloat)

	req := &RenderRequest{
		Latex: latex,
		Scale: scaleFloat,
		Color: color,
	}

	result, err := r.Render(ctx, req)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

// generateHTML 生成 HTML
func (r *Renderer) generateHTML(req *RenderRequest) string {
	var buf bytes.Buffer
	if err := r.template.Execute(&buf, req); err != nil {
		return fmt.Sprintf("<html><body>模板执行错误: %v</body></html>", err)
	}
	return buf.String()
}

// Close 关闭渲染器
func (r *Renderer) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
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
