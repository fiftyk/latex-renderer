// Package latex 提供 LaTeX 公式渲染功能
// 使用 Chrome headless 渲染 LaTeX 公式为 PNG 图片
package latex

import (
	"context"
	"time"

	"github.com/ruxuwu/latex-renderer/cache"
	"github.com/ruxuwu/latex-renderer/renderer"
)

// Config 渲染配置
type Config struct {
	ChromePath string        // Chrome 可执行文件路径
	ChromeArgs string        // Chrome 额外参数
	CacheType  string        // 缓存类型: "local" 或 "oss"
	CacheDir   string        // 本地缓存目录 (CacheType=local 时使用)
	TTL        time.Duration // 缓存过期时间
	// OSS 配置 (CacheType=oss 时使用)
	OSSEndpoint string
	OSSBucket   string
	OSSAccessKey string
	OSSSecretKey string
	OSSDomain   string
}

// Result 渲染结果
type Result struct {
	Data []byte // PNG 图片数据
	URL  string // 缓存 URL (如果有)
}

// Client LaTeX 渲染客户端
type Client struct {
	renderer *renderer.Renderer
	cache    cache.Cache
	ttl      time.Duration
}

// NewClient 创建渲染客户端
func NewClient(cfg *Config) (*Client, error) {
	// 创建渲染器
	r, err := renderer.NewRenderer(cfg.ChromePath, cfg.ChromeArgs)
	if err != nil {
		return nil, err
	}

	// 创建缓存
	var c cache.Cache
	switch cfg.CacheType {
	case "oss":
		c, err = cache.NewOSSCache(&cache.OSSConfig{
			Endpoint:    cfg.OSSEndpoint,
			Bucket:      cfg.OSSBucket,
			AccessKey:   cfg.OSSAccessKey,
			SecretKey:   cfg.OSSSecretKey,
			Domain:      cfg.OSSDomain,
			TTL:         cfg.TTL,
		})
	default:
		c, err = cache.NewLocalCache(&cache.LocalConfig{
			Dir: cfg.CacheDir,
			TTL: cfg.TTL,
		})
	}
	if err != nil {
		r.Close()
		return nil, err
	}

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	return &Client{
		renderer: r,
		cache:    c,
		ttl:      ttl,
	}, nil
}

// Render 渲染 LaTeX 公式为 PNG
func (c *Client) Render(ctx context.Context, latex string, opts ...Option) ([]byte, error) {
	// 应用选项
	o := &options{
		fontSize: "16",
		padding:  "20",
	}
	for _, opt := range opts {
		opt(o)
	}

	// 生成缓存 key
	cacheKey := cache.GenerateCacheKey(latex, "png", "1", o.fontSize, o.padding)

	// 尝试从缓存获取
	data, err := c.cache.Get(ctx, cacheKey)
	if err == nil && data != nil {
		return data, nil
	}

	// 渲染
	data, err = c.renderer.RenderToPNG(ctx, &renderer.RenderOptions{
		Latex:    latex,
		FontSize: o.fontSize,
		Padding:  o.padding,
	})
	if err != nil {
		return nil, err
	}

	// 写入缓存
	_ = c.cache.Set(ctx, cacheKey, data, c.ttl)

	return data, nil
}

// RenderToFile 渲染并保存到文件
func (c *Client) RenderToFile(ctx context.Context, latex, outputPath string, opts ...Option) error {
	data, err := c.Render(ctx, latex, opts...)
	if err != nil {
		return err
	}
	return WriteFile(outputPath, data)
}

// Close 关闭客户端
func (c *Client) Close() error {
	return c.renderer.Close()
}

// option 函数类型
type Option func(*options)

type options struct {
	fontSize string
	padding  string
}

// WithFontSize 设置字体大小
func WithFontSize(size string) Option {
	return func(o *options) {
		o.fontSize = size
	}
}

// WithPadding 设置内边距
func WithPadding(padding string) Option {
	return func(o *options) {
		o.padding = padding
	}
}
