package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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

	// 创建缓存目录（使用0777权限，然后在Docker环境中由chown设置正确权限）
	if err := os.MkdirAll(cfg.Dir, 0777); err != nil {
		log.Printf("[缓存] 创建缓存目录失败: dir=%s, err=%v", cfg.Dir, err)
		return nil, fmt.Errorf("创建缓存目录失败: %w", err)
	}

	// 确保目录可写
	if err := os.Chmod(cfg.Dir, 0777); err != nil {
		log.Printf("[缓存] 设置缓存目录权限失败: dir=%s, err=%v", cfg.Dir, err)
		// 不返回错误，继续尝试，因为可能在容器中已有正确权限
	}

	log.Printf("[缓存] 本地缓存初始化成功: dir=%s, ttl=%v", cfg.Dir, cfg.TTL)

	// 检查目录权限
	if info, err := os.Stat(cfg.Dir); err != nil {
		log.Printf("[缓存] 缓存目录不可访问: dir=%s, err=%v", cfg.Dir, err)
	} else if !info.IsDir() {
		log.Printf("[缓存] 缓存路径不是目录: dir=%s", cfg.Dir)
		return nil, fmt.Errorf("缓存路径不是目录: %s", cfg.Dir)
	} else {
		// 检查目录权限
		if info.Mode()&0002 != 0 {
			log.Printf("[缓存] 缓存目录有写权限: dir=%s, mode=%s", cfg.Dir, info.Mode().String())
		} else {
			log.Printf("[缓存] 警告: 缓存目录可能没有写权限: dir=%s, mode=%s", cfg.Dir, info.Mode().String())
		}
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

	// 读取文件（添加权限检查）
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[缓存-本地] 读取缓存文件失败: key=%s, path=%s, err=%v", key, path, err)
		// 详细错误信息
		if os.IsPermission(err) {
			log.Printf("[缓存-本地] 权限错误详情: 当前用户UID=%v, 文件权限=%v", os.Getuid(), info.Mode().String())
		}
		return nil, fmt.Errorf("读取缓存文件失败: %w", err)
	}

	log.Printf("[缓存-本地] 缓存读取成功: key=%s, size=%d bytes", key, len(data))
	return data, nil
}

// Set 设置缓存
func (c *LocalCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	path := c.getPath(key)
	log.Printf("[缓存-本地] 设置缓存: key=%s, path=%s, size=%d bytes, ttl=%v", key, path, len(data), ttl)

	// 确保目录存在（使用0777权限）
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0777); err != nil {
		log.Printf("[缓存-本地] 创建缓存目录失败: key=%s, dir=%s, err=%v", key, dir, err)
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	// 确保目录有写权限
	if err := os.Chmod(dir, 0777); err != nil {
		log.Printf("[缓存-本地] 设置目录权限失败: key=%s, dir=%s, err=%v", key, dir, err)
		// 继续尝试，不返回错误
	}

	// 写入文件（使用0666权限，允许读写）
	if err := os.WriteFile(path, data, 0666); err != nil {
		log.Printf("[缓存-本地] 写入缓存文件失败: key=%s, path=%s, err=%v", key, path, err)

		// 详细错误信息
		if os.IsPermission(err) {
			log.Printf("[缓存-本地] 权限错误详情: 当前用户UID=%v, 目录权限=%v", os.Getuid(), dir)
		}
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
// 防止路径遍历攻击：确保 key 始终在缓存目录内
func (c *LocalCache) getPath(key string) string {
	// 清理 key 中的路径遍历字符
	key = filepath.ToSlash(key)
	key = strings.TrimPrefix(key, "/")
	// 移除所有 .. 分量
	parts := strings.Split(key, "/")
	var cleanParts []string
	for _, part := range parts {
		if part == ".." {
			continue
		}
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	cleanKey := strings.Join(cleanParts, "/")

	return filepath.Join(c.dir, cleanKey)
}
