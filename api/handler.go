package api

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ruxuwu/latex-renderer/cache"
	"github.com/ruxuwu/latex-renderer/renderer"
)

// Handler HTTP 处理器
type Handler struct {
	renderer         *renderer.Renderer
	cache            cache.Cache
	ttl              time.Duration
	overloadStrategy renderer.OverloadStrategy
	batchRenderer    renderer.BatchRendererInterface
	staticBaseURL    string // 静态资源基础URL
}

// NewHandler 创建处理器
func NewHandler(r *renderer.Renderer, c cache.Cache, ttl time.Duration, strategy renderer.OverloadStrategy, batchRenderer renderer.BatchRendererInterface, staticBaseURL ...string) *Handler {
	var baseURL string
	if len(staticBaseURL) > 0 {
		baseURL = staticBaseURL[0]
	}
	return &Handler{
		renderer:         r,
		cache:            c,
		ttl:              ttl,
		overloadStrategy: strategy,
		batchRenderer:    batchRenderer,
		staticBaseURL:    baseURL,
	}
}

// RenderRequest 渲染请求参数
type RenderRequest struct {
	Latex    string `form:"latex" binding:"required"`
	Format   string `form:"format"`
	FontSize string `form:"fontSize"`
	Padding  string `form:"padding"`
}

// RenderResponse 渲染响应
type RenderResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"`
	URL     string `json:"url,omitempty"`
}

// Render 处理渲染请求
func (h *Handler) Render(c *gin.Context) {
	// 检查并发限制
	if !h.overloadStrategy.Handle() {
		c.JSON(http.StatusServiceUnavailable, RenderResponse{
			Success: false,
			Message: h.overloadStrategy.Reject().Error(),
		})
		return
	}
	defer h.overloadStrategy.Release()

	var req RenderRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, RenderResponse{
			Success: false,
			Message: "缺少 latex 参数",
		})
		return
	}

	// 设置默认值
	if req.Format == "" {
		req.Format = "png"
	}
	if req.FontSize == "" {
		req.FontSize = "16"
	}
	if req.Padding == "" {
		req.Padding = "20"
	}

	// 验证 format
	if req.Format != "" && req.Format != "png" {
		c.JSON(http.StatusBadRequest, RenderResponse{
			Success: false,
			Message: "不支持的格式，仅支持 png",
		})
		return
	}

	// 生成缓存 key
	cacheKey := cache.GenerateCacheKey(req.Latex, req.Format, req.FontSize, req.Padding)

	// 尝试从缓存获取
	var data []byte
	var err error
	data, err = h.cache.Get(c.Request.Context(), cacheKey)
	if err != nil {
		log.Printf("[缓存] 读取缓存失败: %s", sanitizeCacheKey(cacheKey))
		c.Writer.Header().Set("X-Cache-Status", "read-error")
	} else if data != nil {
		// 缓存命中
		log.Printf("[缓存] 缓存命中: %s, size=%d bytes", sanitizeCacheKey(cacheKey), len(data))

		// 生成 ETag (基于内容的MD5)
		etag := fmt.Sprintf(`"%x"`, md5.Sum(data))

		// 检查客户端是否已经有最新版本
		if_none_match := c.GetHeader("If-None-Match")
		if if_none_match == etag {
			log.Printf("[缓存] ETag 匹配，返回 304")
			c.Writer.Header().Set("ETag", etag)
			c.Writer.Header().Set("X-Cache-Status", "hit")
			c.Writer.Header().Set("Cache-Control", "public, max-age=31536000")
			c.Writer.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			c.Status(http.StatusNotModified)
			return
		}

		// 返回完整响应
		log.Printf("[缓存] 返回缓存内容: %s", sanitizeCacheKey(cacheKey))
		c.Writer.Header().Set("ETag", etag)
		c.Writer.Header().Set("X-Cache-Status", "hit")
		h.writeImage(c, data)
		return
	}

	log.Printf("[缓存] 缓存未命中")

	// 缓存未命中，渲染图片
	if h.batchRenderer != nil {
		// 使用批量渲染
		resultCh, enqueueErr := h.batchRenderer.Enqueue(c.Request.Context(), &renderer.RenderOptions{
			Latex:    req.Latex,
			FontSize: req.FontSize,
			Padding:  req.Padding,
		})
		if enqueueErr != nil {
			log.Printf("[批量] 入队失败: %v", enqueueErr)
			c.JSON(http.StatusServiceUnavailable, RenderResponse{
				Success: false,
				Message: "服务繁忙，请稍后重试",
			})
			return
		}

		// 等待结果
		result := <-resultCh
		data, err = result.Data, result.Err
	} else {
		// 直接渲染
		data, err = h.renderer.RenderToPNG(c.Request.Context(), &renderer.RenderOptions{
			Latex:    req.Latex,
			FontSize: req.FontSize,
			Padding:  req.Padding,
		})
	}

	if err != nil {
		// 记录详细错误日志（内部），LaTeX 可能很长，只显示前 50 字符
		latexPreview := req.Latex
		if len(latexPreview) > 50 {
			latexPreview = latexPreview[:50] + "..."
		}
		log.Printf("[渲染] 渲染失败: latex=%s, err=%v", latexPreview, err)
		// 返回脱敏的错误信息（不暴露内部细节）
		c.JSON(http.StatusInternalServerError, RenderResponse{
			Success: false,
			Message: "渲染失败，请稍后重试",
		})
		return
	}

	log.Printf("[渲染] 渲染成功: %s, size=%d bytes", sanitizeCacheKey(cacheKey), len(data))

	// 写入缓存
	log.Printf("[缓存] 写入缓存: %s", sanitizeCacheKey(cacheKey))
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, h.ttl); err != nil {
		log.Printf("[缓存] 写入缓存失败: %s, err=%v", sanitizeCacheKey(cacheKey), err)
		c.Writer.Header().Set("X-Cache-Status", "write-error")
	} else {
		log.Printf("[缓存] 写入缓存成功: %s", sanitizeCacheKey(cacheKey))
		c.Writer.Header().Set("X-Cache-Status", "miss")
	}

	// 返回图片
	h.writeImage(c, data)
}

// writeImage 写入图片响应
func (h *Handler) writeImage(c *gin.Context, data []byte) {
	// 生成 ETag (基于内容的MD5)
	etag := fmt.Sprintf(`"%x"`, md5.Sum(data))

	c.Writer.Header().Set("Content-Type", "image/png")
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Writer.Header().Set("Cache-Control", "public, max-age=31536000")
	c.Writer.Header().Set("ETag", etag)
	c.Writer.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))

	// 添加 X-Cache-Status 头（如果还没有设置）
	if c.Writer.Header().Get("X-Cache-Status") == "" {
		c.Writer.Header().Set("X-Cache-Status", "generated")
	}

	c.Data(http.StatusOK, "image/png", data)
}

// Health 健康检查
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"cache":  h.cache.Name(),
		"uptime": time.Since(startTime).String(),
	})
}

// Info 获取服务信息
func (h *Handler) Info(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "latex-renderer",
		"version": "1.0.0",
		"cache":   h.cache.Name(),
	})
}

var startTime = time.Now()

// sanitizeCacheKey 对缓存 key 进行脱敏处理，只保留前 8 个字符
func sanitizeCacheKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:8] + "***"
}
