package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ruxuwu/latex-renderer/cache"
	"github.com/ruxuwu/latex-renderer/renderer"
)

// Handler HTTP 处理器
type Handler struct {
	renderer *renderer.Renderer
	cache    cache.Cache
	ttl      time.Duration
}

// NewHandler 创建处理器
func NewHandler(r *renderer.Renderer, c cache.Cache, ttl time.Duration) *Handler {
	return &Handler{
		renderer: r,
		cache:    c,
		ttl:      ttl,
	}
}

// RenderRequest 渲染请求参数
type RenderRequest struct {
	Latex    string `form:"latex" binding:"required"`
	Format   string `form:"format"`
	Scale    string `form:"scale"`
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
	if req.Scale == "" {
		req.Scale = "2"
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
	cacheKey := cache.GenerateCacheKey(req.Latex, req.Format, req.Scale, req.FontSize, req.Padding)

	// 尝试从缓存获取
	data, err := h.cache.Get(c.Request.Context(), cacheKey)
	if err != nil {
		c.Writer.Header().Set("X-Cache", "error")
	} else if data != nil {
		c.Writer.Header().Set("X-Cache", "hit")
		h.writeImage(c, data)
		return
	}

	// 缓存未命中，渲染图片
	data, err = h.renderer.RenderToPNG(c.Request.Context(), &renderer.RenderOptions{
		Latex:    req.Latex,
		FontSize: req.FontSize,
		Padding:  req.Padding,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, RenderResponse{
			Success: false,
			Message: fmt.Sprintf("渲染失败: %v", err),
		})
		return
	}

	// 写入缓存
	if err := h.cache.Set(c.Request.Context(), cacheKey, data, h.ttl); err != nil {
		c.Writer.Header().Set("X-Cache", "write-error")
	} else {
		c.Writer.Header().Set("X-Cache", "miss")
	}

	// 返回图片
	h.writeImage(c, data)
}

// writeImage 写入图片响应
func (h *Handler) writeImage(c *gin.Context, data []byte) {
	c.Writer.Header().Set("Content-Type", "image/png")
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Writer.Header().Set("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, "image/png", data)
}

// Health 健康检查
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"cache":   h.cache.Name(),
		"uptime":  time.Since(startTime).String(),
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
