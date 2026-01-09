package renderer

import "time"

// RenderOptions 渲染选项
type RenderOptions struct {
	Latex      string // LaTeX 公式
	Color      string // 字体颜色 (默认 black)
	Background string // 背景颜色 (默认 transparent)
	FontSize   string // 字体大小 px (默认 16)
	Padding    string // 内边距 px (默认 20)
}

// RendererOptions 渲染器配置选项
type RendererOptions struct {
	ExecPath      string        // Chrome 可执行文件路径
	Args          string        // Chrome 启动参数
	MaxConcurrent int           // 最大并发数
	MaxRequests   int64         // 每多少个请求后重启浏览器
	MaxInterval   time.Duration // 最大间隔时间后重启浏览器
	RenderTimeout time.Duration // 单次渲染超时时间
	MaxRetries    int           // 渲染失败最大重试次数
	StaticBaseURL string        // 静态资源基础URL
	Strategy      OverloadStrategy // 并发控制策略（可选）
}
