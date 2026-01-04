package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LocalCache 本地文件系统缓存实现
type LocalCache struct {
	dir string
	ttl time.Duration
}

// NewLocalCache 创建本地缓存实例
func NewLocalCache(cfg LocalConfig) (*LocalCache, error) {
	if cfg.Dir == "" {
		cfg.Dir = "./cache"
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 168 * time.Hour // 默认 7 天
	}

	// 创建缓存目录
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("创建缓存目录失败: %w", err)
	}

	return &LocalCache{
		dir: cfg.Dir,
		ttl: cfg.TTL,
	}, nil
}

// Name 返回缓存名称
func (c *LocalCache) Name() string {
	return "local"
}

// Get 获取缓存
func (c *LocalCache) Get(ctx context.Context, key string) ([]byte, error) {
	path := c.getPath(key)

	// 检查文件是否存在
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil // 缓存未命中
	}
	if err != nil {
		return nil, fmt.Errorf("检查缓存文件失败: %w", err)
	}

	// 检查是否过期
	if time.Since(info.ModTime()) > c.ttl {
		// 删除过期缓存
		os.Remove(path)
		return nil, nil
	}

	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取缓存文件失败: %w", err)
	}

	return data, nil
}

// Set 设置缓存
func (c *LocalCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	path := c.getPath(key)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入缓存文件失败: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (c *LocalCache) Delete(ctx context.Context, key string) error {
	path := c.getPath(key)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("删除缓存文件失败: %w", err)
	}
	return nil
}

// Exists 检查缓存是否存在
func (c *LocalCache) Exists(ctx context.Context, key string) (bool, error) {
	path := c.getPath(key)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetURL 返回本地文件路径 (可用于直接访问)
func (c *LocalCache) GetURL(ctx context.Context, key string) (string, error) {
	path := c.getPath(key)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}
	return path, nil
}

// getPath 获取文件路径
func (c *LocalCache) getPath(key string) string {
	return filepath.Join(c.dir, key)
}
