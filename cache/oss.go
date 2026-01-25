package cache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSCache OSS 缓存实现 (支持阿里云 OSS、腾讯云 COS 等 S3 兼容存储)
type OSSCache struct {
	client     *oss.Client
	bucket     *oss.Bucket
	domain     string
	prefix     string
	ttl        time.Duration
	endpoint   string
	bucketName string
}

// NewOSSCache 创建 OSS 缓存实例
func NewOSSCache(cfg *OSSConfig) (*OSSCache, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("OSS endpoint 不能为空")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("OSS bucket 不能为空")
	}
	if cfg.AccessKey == "" {
		return nil, fmt.Errorf("OSS access_key 不能为空")
	}
	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("OSS secret_key 不能为空")
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 168 * time.Hour // 默认 7 天
	}

	// 创建 OSS 客户端
	client, err := oss.New(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("创建 OSS 客户端失败: %w", err)
	}

	// 获取 bucket
	bucket, err := client.Bucket(cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("获取 bucket 失败: %w", err)
	}

	return &OSSCache{
		client:     client,
		bucket:     bucket,
		domain:     cfg.Domain,
		prefix:     cfg.Prefix,
		ttl:        cfg.TTL,
		endpoint:   cfg.Endpoint,
		bucketName: cfg.Bucket,
	}, nil
}

// Name 返回缓存名称
func (c *OSSCache) Name() string {
	return "oss"
}

// buildKey 构建带前缀的存储 key
func (c *OSSCache) buildKey(key string) string {
	if c.prefix == "" {
		return key
	}
	// 确保前缀以 / 结尾
	prefix := strings.TrimSuffix(c.prefix, "/") + "/"
	return prefix + key
}

// Get 获取缓存
func (c *OSSCache) Get(ctx context.Context, key string) ([]byte, error) {
	ossKey := c.buildKey(key)
	// 检查对象是否存在
	exist, err := c.bucket.IsObjectExist(ossKey)
	if err != nil {
		return nil, fmt.Errorf("检查 OSS 对象失败: %w", err)
	}
	if !exist {
		return nil, nil // 缓存未命中
	}

	// 获取对象元数据
	props, err := c.bucket.GetObjectDetailedMeta(ossKey)
	if err != nil {
		return nil, fmt.Errorf("获取 OSS 对象元数据失败: %w", err)
	}

	// 检查是否过期（TTL <= 0 表示永不过期）
	if c.ttl > 0 {
		if lastModified := props.Get("Last-Modified"); lastModified != "" {
			t, err := time.Parse(http.TimeFormat, lastModified)
			if err == nil && time.Since(t) > c.ttl {
				// 删除过期缓存
				c.bucket.DeleteObject(ossKey)
				return nil, nil
			}
		}
	}

	// 下载对象
	data, err := c.bucket.GetObject(ossKey)
	if err != nil {
		return nil, fmt.Errorf("下载 OSS 对象失败: %w", err)
	}
	defer data.Close()

	// 读取数据
	content, err := io.ReadAll(data)
	if err != nil {
		return nil, fmt.Errorf("读取 OSS 数据失败: %w", err)
	}

	return content, nil
}

// Set 设置缓存
func (c *OSSCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	ossKey := c.buildKey(key)

	var opts []oss.Option
	// 只有 TTL > 0 才设置 OSS 对象过期时间
	if ttl > 0 {
		expiry := time.Now().Add(ttl)
		opts = append(opts, oss.Expires(expiry))
	}
	opts = append(opts, oss.ContentType("image/png"))

	// 上传对象
	err := c.bucket.PutObject(ossKey, bytes.NewReader(data), opts...)
	if err != nil {
		return fmt.Errorf("上传 OSS 对象失败: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (c *OSSCache) Delete(ctx context.Context, key string) error {
	ossKey := c.buildKey(key)
	err := c.bucket.DeleteObject(ossKey)
	if err != nil {
		return fmt.Errorf("删除 OSS 对象失败: %w", err)
	}
	return nil
}

// Exists 检查缓存是否存在
func (c *OSSCache) Exists(ctx context.Context, key string) (bool, error) {
	ossKey := c.buildKey(key)
	exist, err := c.bucket.IsObjectExist(ossKey)
	if err != nil {
		return false, fmt.Errorf("检查 OSS 对象失败: %w", err)
	}
	return exist, nil
}

// GetURL 返回 OSS 访问 URL
func (c *OSSCache) GetURL(ctx context.Context, key string) (string, error) {
	ossKey := c.buildKey(key)
	// 如果配置了自定义 domain，使用自定义 domain
	if c.domain != "" {
		u := fmt.Sprintf("%s/%s", strings.TrimSuffix(c.domain, "/"), ossKey)
		return u, nil
	}

	// 否则生成签名 URL (临时访问链接)
	signedURL, err := c.bucket.SignURL(ossKey, "GET", 3600) // 1小时有效
	if err != nil {
		return "", fmt.Errorf("生成签名 URL 失败: %w", err)
	}

	return signedURL, nil
}

// GetPublicURL 返回公开访问 URL (需要 bucket 设置为公开读取)
func (c *OSSCache) GetPublicURL(key string) string {
	ossKey := c.buildKey(key)
	if c.domain != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.domain, "/"), ossKey)
	}
	// 默认的 OSS 外网地址
	return fmt.Sprintf("https://%s.%s/%s", c.bucketName, c.endpoint, ossKey)
}
