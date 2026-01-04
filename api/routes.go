package api

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, handler *Handler) {
	// 健康检查
	r.GET("/health", handler.Health)

	// 服务信息
	r.GET("/info", handler.Info)

	// API 路由组
	api := r.Group("/api")
	{
		api.GET("", handler.Render)
	}
}
