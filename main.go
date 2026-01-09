package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ruxuwu/latex-renderer/api"
	"github.com/ruxuwu/latex-renderer/cache"
	"github.com/ruxuwu/latex-renderer/config"
	"github.com/ruxuwu/latex-renderer/logger"
	"github.com/ruxuwu/latex-renderer/renderer"
)

func main() {
	// 加载配置
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		return
	}

	// 初始化日志
	log, err := logger.New(&logger.Config{
		Path:    cfg.Log.Path,
		Level:   logger.Level(cfg.Log.Level),
		MaxSize: cfg.Log.MaxSize,
	})
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		return
	}

	log.Print("启动 latex-renderer 服务...")

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

	// 创建并发限制策略（默认 2 个并发）
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}
	log.Printf("最大并发数: %d", maxConcurrent)
	overloadStrategy := renderer.NewFailFastStrategy(maxConcurrent)

	// 启动静态文件HTTP服务器（用于KaTeX资源）
	staticServerPort := 9090
	staticAddr := fmt.Sprintf("127.0.0.1:%d", staticServerPort)
	staticServer := &http.Server{
		Addr:         staticAddr,
		Handler:      http.FileServer(http.Dir("./static")),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("启动静态文件服务器: %s", staticAddr)
		if err := staticServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动静态文件服务器失败: %v", err)
		}
	}()

	// 等待静态服务器启动
	time.Sleep(500 * time.Millisecond)

	staticBaseURL := fmt.Sprintf("http://%s", staticAddr)

	// 初始化渲染器（传入静态资源URL）
	r, err := renderer.NewRenderer(cfg.Chrome.ExecutablePath, cfg.Chrome.Args, maxConcurrent, staticBaseURL)
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
	handler := api.NewHandler(r, cacheImpl, cfg.Cache.TTL, overloadStrategy, staticBaseURL)

	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.LoggerWithWriter(log.Writer()))

	// 设置 API 路由
	api.SetupRoutes(router, handler)

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("启动服务器: %s", addr)
	log.Printf("API: http://%s/api?latex=公式", addr)
	log.Printf("健康检查: http://%s/health", addr)

	// 创建中断信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 启动主服务器
	serverErr := make(chan error, 1)
	go func() {
		log.Println("服务已启动")
		if err := router.Run(addr); err != nil {
			serverErr <- err
		}
	}()

	// 等待中断或服务器错误
	select {
	case <-quit:
		log.Println("正在关闭服务器...")
	case err := <-serverErr:
		log.Fatalf("服务器错误: %v", err)
	}

	// 优雅关闭服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 关闭静态文件服务器
	log.Println("关闭静态文件服务器...")
	if err := staticServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("关闭静态文件服务器失败: %v", err)
	}

	log.Println("服务器已关闭")
}
