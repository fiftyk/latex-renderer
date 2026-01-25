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
		log.Println("正在初始化 OSS 缓存...")
		cacheImpl, err = cache.NewOSSCache(&cfg.Cache.OSS)
		if err != nil {
			log.Fatalf("初始化 OSS 缓存失败: %v", err)
		}
		log.Printf("✓ 使用 OSS 缓存: %s", cfg.Cache.OSS.Endpoint)
	default:
		// 默认使用本地缓存
		log.Println("正在初始化本地缓存...")
		cacheImpl, err = cache.NewLocalCache(&cfg.Cache.Local)
		if err != nil {
			log.Fatalf("初始化本地缓存失败: %v", err)
		}
		log.Printf("✓ 使用本地缓存: %s", cfg.Cache.Local.Dir)
		log.Printf("✓ 缓存 TTL: %v", cfg.Cache.TTL)
	}

	// 查找 Chrome
	chromePath := renderer.FindChrome()
	if chromePath == "" {
		log.Println("警告: 未找到 Chrome，尝试使用系统默认")
	}

	// 创建并发限制策略
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	log.Printf("最大并发数: %d", maxConcurrent)
	log.Printf("浏览器重启阈值: %d 请求 或 %v", cfg.Renderer.MaxRequests, cfg.Renderer.MaxInterval)
	log.Printf("渲染超时: %v, 最大重试: %d 次", cfg.Renderer.RenderTimeout, cfg.Renderer.MaxRetries)

	// 根据配置选择过载策略
	var overloadStrategy renderer.OverloadStrategy
	switch cfg.Renderer.OverloadStrategy {
	case "failfast":
		log.Printf("过载策略: FailFast (快速失败)")
		overloadStrategy = renderer.NewFailFastStrategy(maxConcurrent)
	case "queue":
		log.Printf("过载策略: TimeoutQueue (超时排队)")
		log.Printf("队列大小: %d, 排队超时: %v", cfg.Renderer.QueueSize, cfg.Renderer.QueueTimeout)
		overloadStrategy = renderer.NewTimeoutQueueStrategy(maxConcurrent, cfg.Renderer.QueueSize, cfg.Renderer.QueueTimeout)
	default:
		log.Printf("过载策略: FailFast (未知策略 '%s'，使用默认)", cfg.Renderer.OverloadStrategy)
		overloadStrategy = renderer.NewFailFastStrategy(maxConcurrent)
	}

	// 批量渲染配置
	var batchRenderer renderer.BatchRendererInterface
	if cfg.Batch.Enabled {
		log.Printf("批量渲染: 已启用, 批量大小=%d, 时间窗口=%v, 队列大小=%d",
			cfg.Batch.BatchSize, cfg.Batch.BatchWindow, cfg.Batch.QueueSize)
	} else {
		log.Printf("批量渲染: 未启用")
	}

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

	// 等待静态服务器启动（使用健康检查而非固定延迟）
	staticBaseURL := fmt.Sprintf("http://%s", staticAddr)
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(staticBaseURL + "/katex/katex.min.css")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Printf("静态文件服务器已启动: %s", staticBaseURL)
				break
			}
		}
		if i == maxRetries-1 {
			log.Printf("警告: 静态文件服务器启动超时，继续尝试...")
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 初始化渲染器（传入配置）
	r, err := renderer.NewRenderer(&renderer.RendererOptions{
		ExecPath:      cfg.Chrome.ExecutablePath,
		Args:          cfg.Chrome.Args,
		MaxConcurrent: maxConcurrent,
		MaxRequests:   cfg.Renderer.MaxRequests,
		MaxInterval:   cfg.Renderer.MaxInterval,
		RenderTimeout: cfg.Renderer.RenderTimeout,
		MaxRetries:    cfg.Renderer.MaxRetries,
		StaticBaseURL: staticBaseURL,
		Strategy:      overloadStrategy,
	})
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

	// 初始化批量渲染器（需要在实际渲染器之后）
	if cfg.Batch.Enabled {
		batchConfig := &renderer.BatchConfig{
			BatchSize:   cfg.Batch.BatchSize,
			BatchWindow: cfg.Batch.BatchWindow,
			QueueSize:   cfg.Batch.QueueSize,
		}
		batchRenderer = renderer.NewBatchRenderer(r, batchConfig)
		batchRenderer.Start()
	}

	// 初始化处理器
	handler := api.NewHandler(r, cacheImpl, cfg.Cache.TTL, overloadStrategy, batchRenderer, staticBaseURL)

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

	// 关闭批量渲染器
	if batchRenderer != nil {
		log.Println("关闭批量渲染器...")
		if err := batchRenderer.Stop(); err != nil {
			log.Printf("关闭批量渲染器失败: %v", err)
		}
	}

	log.Println("服务器已关闭")
}
