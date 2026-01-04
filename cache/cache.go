package cache

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"
)

// Cache 缓存接口定义
type Cache interface {
	// Get 获取缓存，返回缓存数据
	Get(ctx context.Context, key string) ([]byte, error)
	// Set 设置缓存，data 为图片二进制数据
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)
	// GetURL 返回缓存的访问 URL (OSS 等云存储需要)
	GetURL(ctx context.Context, key string) (string, error)
	// Name 返回缓存类型名称
	Name() string
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type string      `yaml:"type" env:"CACHE_TYPE"`
	TTL  time.Duration `yaml:"ttl" env:"CACHE_TTL"`
}

// LocalConfig 本地缓存配置
type LocalConfig struct {
	Dir string        `yaml:"dir" env:"CACHE_LOCAL_DIR"`
	TTL time.Duration `yaml:"ttl" env:"CACHE_LOCAL_TTL"`
}

// OSSConfig OSS 缓存配置
type OSSConfig struct {
	Endpoint    string `yaml:"endpoint" env:"OSS_ENDPOINT"`
	Bucket      string `yaml:"bucket" env:"OSS_BUCKET"`
	AccessKey   string `yaml:"access_key" env:"OSS_ACCESS_KEY"`
	SecretKey   string `yaml:"secret_key" env:"OSS_SECRET_KEY"`
	Domain      string `yaml:"domain" env:"OSS_DOMAIN"`
	TTL         time.Duration `yaml:"ttl" env:"OSS_TTL"`
}

// GenerateCacheKey 生成缓存 key
func GenerateCacheKey(latex, format, scale, color, background, fontSize, padding string) string {
	content := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", latex, format, scale, color, background, fontSize, padding)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	return fmt.Sprintf("latex/%s.%s", hash, format)
}
