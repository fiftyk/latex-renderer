package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/ruxuwu/latex-renderer/api"
	"github.com/ruxuwu/latex-renderer/cache"
	"github.com/ruxuwu/latex-renderer/config"
	"github.com/ruxuwu/latex-renderer/renderer"
)

func main() {
	// 加载配置
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化缓存
	var cacheImpl cache.Cache
	switch cfg.Cache.Type {
	case "oss":
		cacheImpl, err = cache.NewOSSCache(&cfg.Cache.OSS)
		if err != nil {
			log.Fatalf("初始化 OSS 缓存失败: %v", err)
		}
		log.Printf("使用 OSS 缓存: %s", cfg.Cache.OSS.Endpoint)
	default:
		// 默认使用本地缓存
		cacheImpl, err = cache.NewLocalCache(&cfg.Cache.Local)
		if err != nil {
			log.Fatalf("初始化本地缓存失败: %v", err)
		}
		log.Printf("使用本地缓存: %s", cfg.Cache.Local.Dir)
	}

	// 查找 Chrome
	chromePath := renderer.FindChrome()
	if chromePath == "" {
		log.Println("警告: 未找到 Chrome，尝试使用系统默认")
	}

	// 初始化渲染器
	r, err := renderer.NewRenderer(cfg.Chrome.ExecutablePath, cfg.Chrome.Args)
	if err != nil {
		log.Fatalf("初始化渲染器失败: %v", err)
	}
	defer r.Close()

	// 预热 Chrome 浏览器
	log.Println("正在预热 Chrome 浏览器...")
	if err := r.Warmup(context.Background()); err != nil {
		log.Printf("警告: 预热 Chrome 失败: %v", err)
	} else {
		log.Println("Chrome 浏览器预热完成")
	}

	// 初始化处理器
	handler := api.NewHandler(r, cacheImpl, cfg.Cache.TTL)

	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// 设置 API 路由
	api.SetupRoutes(router, handler)

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("启动服务器: %s", addr)
	log.Printf("API: http://%s/api?latex=公式", addr)
	log.Printf("健康检查: http://%s/health", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
