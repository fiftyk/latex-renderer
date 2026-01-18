package config

import (
	"os"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	// 验证默认值
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port should be 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host should be '0.0.0.0', got %s", cfg.Server.Host)
	}
	if cfg.Cache.Type != "local" {
		t.Errorf("Cache.Type should be 'local', got %s", cfg.Cache.Type)
	}
	if cfg.Cache.TTL != 168*time.Hour {
		t.Errorf("Cache.TTL should be 168h, got %v", cfg.Cache.TTL)
	}
	if cfg.Cache.Local.Dir != "./cache" {
		t.Errorf("Cache.Local.Dir should be './cache', got %s", cfg.Cache.Local.Dir)
	}
	if cfg.Chrome.Args == "" {
		t.Errorf("Chrome.Args should not be empty")
	}
	// 验证日志默认值
	if cfg.Log.MaxSize != 100 {
		t.Errorf("Log.MaxSize should be 100, got %d", cfg.Log.MaxSize)
	}
	if cfg.Log.MaxFiles != 3 {
		t.Errorf("Log.MaxFiles should be 3, got %d", cfg.Log.MaxFiles)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level should be 'info', got %s", cfg.Log.Level)
	}
	// 验证并发默认值
	if cfg.MaxConcurrent != 2 {
		t.Errorf("MaxConcurrent should be 2, got %d", cfg.MaxConcurrent)
	}
	// 验证渲染器默认值
	if cfg.Renderer.MaxRequests != 50 {
		t.Errorf("Renderer.MaxRequests should be 50, got %d", cfg.Renderer.MaxRequests)
	}
	if cfg.Renderer.MaxInterval != 10*time.Minute {
		t.Errorf("Renderer.MaxInterval should be 10m, got %v", cfg.Renderer.MaxInterval)
	}
	if cfg.Renderer.RenderTimeout != 30*time.Second {
		t.Errorf("Renderer.RenderTimeout should be 30s, got %v", cfg.Renderer.RenderTimeout)
	}
	if cfg.Renderer.MaxRetries != 2 {
		t.Errorf("Renderer.MaxRetries should be 2, got %d", cfg.Renderer.MaxRetries)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// 设置环境变量
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("SERVER_HOST", "localhost")
	os.Setenv("CACHE_TYPE", "oss")
	os.Setenv("CACHE_TTL", "24h")
	os.Setenv("OSS_ENDPOINT", "oss-cn-hangzhou.aliyuncs.com")
	os.Setenv("OSS_BUCKET", "test-bucket")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("CACHE_TYPE")
		os.Unsetenv("CACHE_TTL")
		os.Unsetenv("OSS_ENDPOINT")
		os.Unsetenv("OSS_BUCKET")
	}()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv 失败: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port should be 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("Server.Host should be 'localhost', got %s", cfg.Server.Host)
	}
	if cfg.Cache.Type != "oss" {
		t.Errorf("Cache.Type should be 'oss', got %s", cfg.Cache.Type)
	}
	if cfg.Cache.TTL != 24*time.Hour {
		t.Errorf("Cache.TTL should be 24h, got %v", cfg.Cache.TTL)
	}
	if cfg.Cache.OSS.Endpoint != "oss-cn-hangzhou.aliyuncs.com" {
		t.Errorf("OSS.Endpoint should be 'oss-cn-hangzhou.aliyuncs.com', got %s", cfg.Cache.OSS.Endpoint)
	}
	if cfg.Cache.OSS.Bucket != "test-bucket" {
		t.Errorf("OSS.Bucket should be 'test-bucket', got %s", cfg.Cache.OSS.Bucket)
	}
}

func TestLoadFromEnv_LocalCache(t *testing.T) {
	os.Setenv("CACHE_LOCAL_DIR", "/custom/cache")
	os.Setenv("CACHE_LOCAL_TTL", "48h")
	defer func() {
		os.Unsetenv("CACHE_LOCAL_DIR")
		os.Unsetenv("CACHE_LOCAL_TTL")
	}()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv 失败: %v", err)
	}

	if cfg.Cache.Local.Dir != "/custom/cache" {
		t.Errorf("Cache.Local.Dir should be '/custom/cache', got %s", cfg.Cache.Local.Dir)
	}
	if cfg.Cache.Local.TTL != 48*time.Hour {
		t.Errorf("Cache.Local.TTL should be 48h, got %v", cfg.Cache.Local.TTL)
	}
}

func TestLoadFromEnv_ChromeConfig(t *testing.T) {
	os.Setenv("CHROME_EXECUTABLE_PATH", "/usr/bin/chromium")
	os.Setenv("CHROME_ARGS", "--no-sandbox")
	defer func() {
		os.Unsetenv("CHROME_EXECUTABLE_PATH")
		os.Unsetenv("CHROME_ARGS")
	}()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv 失败: %v", err)
	}

	if cfg.Chrome.ExecutablePath != "/usr/bin/chromium" {
		t.Errorf("Chrome.ExecutablePath should be '/usr/bin/chromium', got %s", cfg.Chrome.ExecutablePath)
	}
	if cfg.Chrome.Args != "--no-sandbox" {
		t.Errorf("Chrome.Args should be '--no-sandbox', got %s", cfg.Chrome.Args)
	}
}

func TestLoadFromEnv_LogConfig(t *testing.T) {
	os.Setenv("LOG_PATH", "/var/log/latex-renderer/app.log")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("LOG_PATH")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv 失败: %v", err)
	}

	if cfg.Log.Path != "/var/log/latex-renderer/app.log" {
		t.Errorf("Log.Path should be '/var/log/latex-renderer/app.log', got %s", cfg.Log.Path)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level should be 'debug', got %s", cfg.Log.Level)
	}
}

func TestLoadFromEnv_RendererConfig(t *testing.T) {
	os.Setenv("RENDERER_MAX_REQUESTS", "200")
	os.Setenv("RENDERER_MAX_INTERVAL", "1h")
	os.Setenv("RENDERER_TIMEOUT", "60s")
	os.Setenv("RENDERER_MAX_RETRIES", "3")
	os.Setenv("MAX_CONCURRENT", "8")
	defer func() {
		os.Unsetenv("RENDERER_MAX_REQUESTS")
		os.Unsetenv("RENDERER_MAX_INTERVAL")
		os.Unsetenv("RENDERER_TIMEOUT")
		os.Unsetenv("RENDERER_MAX_RETRIES")
		os.Unsetenv("MAX_CONCURRENT")
	}()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv 失败: %v", err)
	}

	if cfg.Renderer.MaxRequests != 200 {
		t.Errorf("Renderer.MaxRequests should be 200, got %d", cfg.Renderer.MaxRequests)
	}
	if cfg.Renderer.MaxInterval != 1*time.Hour {
		t.Errorf("Renderer.MaxInterval should be 1h, got %v", cfg.Renderer.MaxInterval)
	}
	if cfg.Renderer.RenderTimeout != 60*time.Second {
		t.Errorf("Renderer.RenderTimeout should be 60s, got %v", cfg.Renderer.RenderTimeout)
	}
	if cfg.Renderer.MaxRetries != 3 {
		t.Errorf("Renderer.MaxRetries should be 3, got %d", cfg.Renderer.MaxRetries)
	}
	if cfg.MaxConcurrent != 8 {
		t.Errorf("MaxConcurrent should be 8, got %d", cfg.MaxConcurrent)
	}
}
