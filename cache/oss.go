package cache

import (
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
		ttl:        cfg.TTL,
		endpoint:   cfg.Endpoint,
		bucketName: cfg.Bucket,
	}, nil
}

// Name 返回缓存名称
func (c *OSSCache) Name() string {
	return "oss"
}

// Get 获取缓存
func (c *OSSCache) Get(ctx context.Context, key string) ([]byte, error) {
	// 检查对象是否存在
	exist, err := c.bucket.IsObjectExist(key)
	if err != nil {
		return nil, fmt.Errorf("检查 OSS 对象失败: %w", err)
	}
	if !exist {
		return nil, nil // 缓存未命中
	}

	// 获取对象元数据
	props, err := c.bucket.GetObjectDetailedMeta(key)
	if err != nil {
		return nil, fmt.Errorf("获取 OSS 对象元数据失败: %w", err)
	}

	// 检查是否过期
	if lastModified := props.Get("Last-Modified"); lastModified != "" {
		t, err := time.Parse(http.TimeFormat, lastModified)
		if err == nil && time.Since(t) > c.ttl {
			// 删除过期缓存
			c.bucket.DeleteObject(key)
			return nil, nil
		}
	}

	// 下载对象
	data, err := c.bucket.GetObject(key)
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
	// 设置过期时间
	expiry := time.Now().Add(ttl)
	opts := []oss.Option{
		oss.Expires(expiry),
		oss.ContentType("image/png"),
	}

	// 上传对象
	err := c.bucket.PutObject(key, strings.NewReader(string(data)), opts...)
	if err != nil {
		return fmt.Errorf("上传 OSS 对象失败: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (c *OSSCache) Delete(ctx context.Context, key string) error {
	err := c.bucket.DeleteObject(key)
	if err != nil {
		return fmt.Errorf("删除 OSS 对象失败: %w", err)
	}
	return nil
}

// Exists 检查缓存是否存在
func (c *OSSCache) Exists(ctx context.Context, key string) (bool, error) {
	exist, err := c.bucket.IsObjectExist(key)
	if err != nil {
		return false, fmt.Errorf("检查 OSS 对象失败: %w", err)
	}
	return exist, nil
}

// GetURL 返回 OSS 访问 URL
func (c *OSSCache) GetURL(ctx context.Context, key string) (string, error) {
	// 如果配置了自定义 domain，使用自定义 domain
	if c.domain != "" {
		u := fmt.Sprintf("%s/%s", strings.TrimSuffix(c.domain, "/"), key)
		return u, nil
	}

	// 否则生成签名 URL (临时访问链接)
	signedURL, err := c.bucket.SignURL(key, "GET", 3600) // 1小时有效
	if err != nil {
		return "", fmt.Errorf("生成签名 URL 失败: %w", err)
	}

	return signedURL, nil
}

// GetPublicURL 返回公开访问 URL (需要 bucket 设置为公开读取)
func (c *OSSCache) GetPublicURL(key string) string {
	if c.domain != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.domain, "/"), key)
	}
	// 默认的 OSS 外网地址
	return fmt.Sprintf("https://%s.%s/%s", c.bucketName, c.endpoint, key)
}
