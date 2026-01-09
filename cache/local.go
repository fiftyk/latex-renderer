package cache

import (
	"context"
	"fmt"
	"log"
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
func NewLocalCache(cfg *LocalConfig) (*LocalCache, error) {
	if cfg.Dir == "" {
		cfg.Dir = "./cache"
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 168 * time.Hour // 默认 7 天
	}

	// 创建缓存目录
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		log.Printf("[缓存] 创建缓存目录失败: dir=%s, err=%v", cfg.Dir, err)
		return nil, fmt.Errorf("创建缓存目录失败: %w", err)
	}

	log.Printf("[缓存] 本地缓存初始化成功: dir=%s, ttl=%v", cfg.Dir, cfg.TTL)

	// 检查目录权限
	if info, err := os.Stat(cfg.Dir); err != nil {
		log.Printf("[缓存] 缓存目录不可访问: dir=%s, err=%v", cfg.Dir, err)
	} else if !info.IsDir() {
		log.Printf("[缓存] 缓存路径不是目录: dir=%s", cfg.Dir)
		return nil, fmt.Errorf("缓存路径不是目录: %s", cfg.Dir)
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
	log.Printf("[缓存-本地] 获取缓存: key=%s, path=%s", key, path)

	// 检查文件是否存在
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Printf("[缓存-本地] 缓存不存在: key=%s, path=%s", key, path)
		return nil, nil // 缓存未命中
	}
	if err != nil {
		log.Printf("[缓存-本地] 检查缓存文件失败: key=%s, path=%s, err=%v", key, path, err)
		return nil, fmt.Errorf("检查缓存文件失败: %w", err)
	}

	log.Printf("[缓存-本地] 缓存文件信息: key=%s, path=%s, size=%d bytes, modTime=%v, ttl=%v", key, path, info.Size(), info.ModTime(), c.ttl)

	// 检查是否过期
	if time.Since(info.ModTime()) > c.ttl {
		log.Printf("[缓存-本地] 缓存已过期: key=%s, age=%v, ttl=%v", key, time.Since(info.ModTime()), c.ttl)
		// 删除过期缓存
		if err := os.Remove(path); err != nil {
			log.Printf("[缓存-本地] 删除过期缓存失败: key=%s, err=%v", key, err)
		} else {
			log.Printf("[缓存-本地] 删除过期缓存成功: key=%s", key)
		}
		return nil, nil
	}

	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[缓存-本地] 读取缓存文件失败: key=%s, path=%s, err=%v", key, path, err)
		return nil, fmt.Errorf("读取缓存文件失败: %w", err)
	}

	log.Printf("[缓存-本地] 缓存读取成功: key=%s, size=%d bytes", key, len(data))
	return data, nil
}

// Set 设置缓存
func (c *LocalCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	path := c.getPath(key)
	log.Printf("[缓存-本地] 设置缓存: key=%s, path=%s, size=%d bytes, ttl=%v", key, path, len(data), ttl)

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[缓存-本地] 创建缓存目录失败: key=%s, dir=%s, err=%v", key, dir, err)
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("[缓存-本地] 写入缓存文件失败: key=%s, path=%s, err=%v", key, path, err)
		return fmt.Errorf("写入缓存文件失败: %w", err)
	}

	log.Printf("[缓存-本地] 写入缓存成功: key=%s, path=%s", key, path)

	// 验证文件是否真的写入成功
	if info, err := os.Stat(path); err != nil {
		log.Printf("[缓存-本地] 验证写入失败: key=%s, path=%s, err=%v", key, path, err)
	} else {
		log.Printf("[缓存-本地] 验证写入成功: key=%s, path=%s, actualSize=%d bytes", key, path, info.Size())
	}

	return nil
}

// Delete 删除缓存
func (c *LocalCache) Delete(ctx context.Context, key string) error {
	path := c.getPath(key)
	log.Printf("[缓存-本地] 删除缓存: key=%s, path=%s", key, path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("[缓存-本地] 缓存不存在，无需删除: key=%s", key)
		return nil
	}
	if err := os.Remove(path); err != nil {
		log.Printf("[缓存-本地] 删除缓存文件失败: key=%s, path=%s, err=%v", key, path, err)
		return fmt.Errorf("删除缓存文件失败: %w", err)
	}
	log.Printf("[缓存-本地] 删除缓存成功: key=%s", key)
	return nil
}

// Exists 检查缓存是否存在
func (c *LocalCache) Exists(ctx context.Context, key string) (bool, error) {
	path := c.getPath(key)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Printf("[缓存-本地] 缓存不存在: key=%s, path=%s", key, path)
		return false, nil
	}
	if err != nil {
		log.Printf("[缓存-本地] 检查缓存失败: key=%s, path=%s, err=%v", key, path, err)
		return false, err
	}
	log.Printf("[缓存-本地] 缓存存在: key=%s, path=%s", key, path)
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
