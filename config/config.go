package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/ruxuwu/latex-renderer/cache"
)

// Config 应用配置
type Config struct {
	Server        ServerConfig   `mapstructure:"server"`
	Cache         cacheConfig    `mapstructure:"cache"`
	Chrome        ChromeConfig   `mapstructure:"chrome"`
	Renderer      RendererConfig `mapstructure:"renderer"`
	Log           LogConfig      `mapstructure:"log"`
	Batch         BatchConfig    `mapstructure:"batch"`
	MaxConcurrent int            `mapstructure:"max_concurrent"` // 最大并发数
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// ChromeConfig Chrome 配置
type ChromeConfig struct {
	ExecutablePath string `mapstructure:"executable_path"`
	Args           string `mapstructure:"args"`
}

// RendererConfig 渲染器配置
type RendererConfig struct {
	MaxRequests      int64         `mapstructure:"max_requests"`      // 每多少个请求后重启浏览器
	MaxInterval      time.Duration `mapstructure:"max_interval"`      // 最大间隔时间后重启浏览器
	RenderTimeout    time.Duration `mapstructure:"render_timeout"`    // 单次渲染超时时间
	MaxRetries       int           `mapstructure:"max_retries"`       // 渲染失败最大重试次数
	QueueSize        int           `mapstructure:"queue_size"`        // 并发队列大小
	QueueTimeout     time.Duration `mapstructure:"queue_timeout"`     // 排队超时时间
	OverloadStrategy string        `mapstructure:"overload_strategy"` // 过载处理策略 (failfast/queue)
}

// LogConfig 日志配置
type LogConfig struct {
	Path     string `mapstructure:"path"`      // 日志文件路径，空则输出到 stdout
	MaxSize  int    `mapstructure:"max_size"`  // 单个日志文件最大尺寸 MB，默认 100
	MaxFiles int    `mapstructure:"max_files"` // 保留的日志文件数量，默认 3
	Level    string `mapstructure:"level"`     // 日志级别: debug, info, warn, error
}

// cacheConfig 缓存配置
type cacheConfig struct {
	Type  string            `mapstructure:"type"`
	TTL   time.Duration     `mapstructure:"ttl"`
	Local cache.LocalConfig `mapstructure:"local"`
	OSS   cache.OSSConfig   `mapstructure:"oss"`
}

// BatchConfig 批量渲染配置
type BatchConfig struct {
	Enabled     bool          `mapstructure:"enabled"`       // 是否启用批量渲染
	BatchSize   int           `mapstructure:"batch_size"`    // 批量大小阈值
	BatchWindow time.Duration `mapstructure:"batch_window"`  // 时间窗口
	QueueSize   int           `mapstructure:"queue_size"`    // 队列缓冲大小
}

// Load 加载配置
func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	// 允许环境变量覆盖
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取配置
	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，使用默认配置
			return Default(), nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 从默认配置开始，确保所有默认值一致
	cfg := Default()
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return cfg, nil
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() (*Config, error) {
	cfg := Default()

	// 从环境变量读取
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &cfg.Server.Port); err != nil {
			// 解析失败，保持默认值
		}
	}
	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("CACHE_TYPE"); v != "" {
		cfg.Cache.Type = v
	}
	if v := os.Getenv("CACHE_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			cfg.Cache.TTL = d
		}
	}

	// OSS 配置
	if v := os.Getenv("OSS_ENDPOINT"); v != "" {
		cfg.Cache.OSS.Endpoint = v
	}
	if v := os.Getenv("OSS_BUCKET"); v != "" {
		cfg.Cache.OSS.Bucket = v
	}
	if v := os.Getenv("OSS_ACCESS_KEY"); v != "" {
		cfg.Cache.OSS.AccessKey = v
	}
	if v := os.Getenv("OSS_SECRET_KEY"); v != "" {
		cfg.Cache.OSS.SecretKey = v
	}
	if v := os.Getenv("OSS_DOMAIN"); v != "" {
		cfg.Cache.OSS.Domain = v
	}
	if v := os.Getenv("OSS_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			cfg.Cache.OSS.TTL = d
		}
	}
	if v := os.Getenv("OSS_PREFIX"); v != "" {
		cfg.Cache.OSS.Prefix = v
	}

	// 本地缓存配置
	if v := os.Getenv("CACHE_LOCAL_DIR"); v != "" {
		cfg.Cache.Local.Dir = v
	}
	if v := os.Getenv("CACHE_LOCAL_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			cfg.Cache.Local.TTL = d
		}
	}

	// Chrome 配置
	if v := os.Getenv("CHROME_EXECUTABLE_PATH"); v != "" {
		cfg.Chrome.ExecutablePath = v
	}
	if v := os.Getenv("CHROME_ARGS"); v != "" {
		cfg.Chrome.Args = v
	}

	// 日志配置
	if v := os.Getenv("LOG_PATH"); v != "" {
		cfg.Log.Path = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}

	// 并发配置
	if v := os.Getenv("MAX_CONCURRENT"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			cfg.MaxConcurrent = n
		}
	}

	// 渲染器配置
	if v := os.Getenv("RENDERER_MAX_REQUESTS"); v != "" {
		var n int64
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			cfg.Renderer.MaxRequests = n
		}
	}
	if v := os.Getenv("RENDERER_MAX_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			cfg.Renderer.MaxInterval = d
		}
	}
	if v := os.Getenv("RENDERER_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			cfg.Renderer.RenderTimeout = d
		}
	}
	if v := os.Getenv("RENDERER_MAX_RETRIES"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n >= 0 {
			cfg.Renderer.MaxRetries = n
		}
	}
	if v := os.Getenv("RENDERER_QUEUE_SIZE"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n >= 0 {
			cfg.Renderer.QueueSize = n
		}
	}
	if v := os.Getenv("RENDERER_QUEUE_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			cfg.Renderer.QueueTimeout = d
		}
	}
	if v := os.Getenv("RENDERER_OVERLOAD_STRATEGY"); v != "" {
		cfg.Renderer.OverloadStrategy = v
	}

	// 批量渲染配置
	if v := os.Getenv("BATCH_ENABLED"); v != "" {
		cfg.Batch.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("BATCH_SIZE"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			cfg.Batch.BatchSize = n
		}
	}
	if v := os.Getenv("BATCH_WINDOW"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			cfg.Batch.BatchWindow = d
		}
	}
	if v := os.Getenv("BATCH_QUEUE_SIZE"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			cfg.Batch.QueueSize = n
		}
	}

	return cfg, nil
}

// Default 返回默认配置
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Cache: cacheConfig{
			Type: "local",
			TTL:  168 * time.Hour,
			Local: cache.LocalConfig{
				Dir: "./cache",
				TTL: 168 * time.Hour,
			},
			OSS: cache.OSSConfig{
				Prefix: "latex/",
				TTL:    168 * time.Hour,
			},
		},
		Chrome: ChromeConfig{
			Args: "--no-sandbox --disable-setuid-sandbox --disable-dev-shm-usage",
		},
		Renderer: RendererConfig{
			MaxRequests:      50,               // 每50个请求重启浏览器（低内存优化）
			MaxInterval:      10 * time.Minute, // 每10分钟重启浏览器（低内存优化）
			RenderTimeout:    30 * time.Second, // 单次渲染超时30秒
			MaxRetries:       2,                // 最多重试2次
			QueueSize:        4,                // 默认队列大小（低内存优化）
			QueueTimeout:     10 * time.Second, // 默认排队超时10秒
			OverloadStrategy: "queue",          // 默认使用排队策略
		},
		Log: LogConfig{
			MaxSize:  100,
			MaxFiles: 3,
			Level:    "info",
		},
		Batch: BatchConfig{
			Enabled:     false,
			BatchSize:   10,
			BatchWindow: 100 * time.Millisecond,
			QueueSize:   100,
		},
		MaxConcurrent: 2, // 默认最多 2 个并发（适合 2GB 服务器）
	}
}
