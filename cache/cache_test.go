package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		name    string
		latex1  string
		scale1  string
		font1   string
		latex2  string
		scale2  string
		font2   string
		want    bool
	}{
		{
			name:   "same formula produces same key",
			latex1: "\\frac{a}{b}",
			scale1: "2",
			font1:  "16",
			latex2: "\\frac{a}{b}",
			scale2: "2",
			font2:  "16",
			want:   true,
		},
		{
			name:   "different formula produces different key",
			latex1: "\\frac{a}{b}",
			scale1: "2",
			font1:  "16",
			latex2: "\\frac{c}{d}",
			scale2: "2",
			font2:  "16",
			want:   false,
		},
		{
			name:   "different scale produces different key",
			latex1: "\\frac{a}{b}",
			scale1: "2",
			font1:  "16",
			latex2: "\\frac{a}{b}",
			scale2: "4",
			font2:  "16",
			want:   false,
		},
		{
			name:   "different fontSize produces different key",
			latex1: "\\frac{a}{b}",
			scale1: "2",
			font1:  "16",
			latex2: "\\frac{a}{b}",
			scale2: "2",
			font2:  "24",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := GenerateCacheKey(tt.latex1, "png", tt.scale1, tt.font1, "20")
			key2 := GenerateCacheKey(tt.latex2, "png", tt.scale2, tt.font2, "20")

			if tt.want {
				if key1 != key2 {
					t.Errorf("GenerateCacheKey() same inputs should produce same key, got %s and %s", key1, key2)
				}
			} else {
				if key1 == key2 {
					t.Errorf("GenerateCacheKey() different inputs should produce different key, got same: %s", key1)
				}
			}
		})
	}
}

func TestLocalCache_SetAndGet(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "latex-cache-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := LocalConfig{
		Dir: tmpDir,
		TTL: 10 * time.Minute,
	}

	cache, err := NewLocalCache(cfg)
	if err != nil {
		t.Fatalf("创建缓存失败: %v", err)
	}

	ctx := context.Background()
	key := "test/formula.png"
	data := []byte("test image data")

	// 测试 Set
	err = cache.Set(ctx, key, data, 10*time.Minute)
	if err != nil {
		t.Fatalf("Set 失败: %v", err)
	}

	// 测试 Get
	result, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}

	if string(result) != string(data) {
		t.Errorf("Get 数据不匹配, got %s, want %s", result, data)
	}
}

func TestLocalCache_Exists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latex-cache-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := LocalConfig{
		Dir: tmpDir,
		TTL: 10 * time.Minute,
	}

	cache, err := NewLocalCache(cfg)
	if err != nil {
		t.Fatalf("创建缓存失败: %v", err)
	}

	ctx := context.Background()
	key := "test/exists.png"

	// 测试不存在的 key
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists 失败: %v", err)
	}
	if exists {
		t.Errorf("Exists() for non-existent key should return false")
	}

	// 设置数据
	err = cache.Set(ctx, key, []byte("data"), 10*time.Minute)
	if err != nil {
		t.Fatalf("Set 失败: %v", err)
	}

	// 测试存在的 key
	exists, err = cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists 失败: %v", err)
	}
	if !exists {
		t.Errorf("Exists() for existing key should return true")
	}
}

func TestLocalCache_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latex-cache-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := LocalConfig{
		Dir: tmpDir,
		TTL: 10 * time.Minute,
	}

	cache, err := NewLocalCache(cfg)
	if err != nil {
		t.Fatalf("创建缓存失败: %v", err)
	}

	ctx := context.Background()
	key := "test/delete.png"

	// 设置数据
	err = cache.Set(ctx, key, []byte("data"), 10*time.Minute)
	if err != nil {
		t.Fatalf("Set 失败: %v", err)
	}

	// 删除数据
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	// 验证已删除
	exists, _ := cache.Exists(ctx, key)
	if exists {
		t.Errorf("Delete 后 Exists 应该返回 false")
	}
}

func TestLocalCache_GetNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latex-cache-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := LocalConfig{
		Dir: tmpDir,
		TTL: 10 * time.Minute,
	}

	cache, err := NewLocalCache(cfg)
	if err != nil {
		t.Fatalf("创建缓存失败: %v", err)
	}

	ctx := context.Background()
	result, err := cache.Get(ctx, "nonexistent/key.png")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}

	if result != nil {
		t.Errorf("Get non-existent key should return nil, got %v", result)
	}
}

func TestLocalCache_Name(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "latex-cache-test")
	defer os.RemoveAll(tmpDir)

	cache, _ := NewLocalCache(LocalConfig{Dir: tmpDir})

	if cache.Name() != "local" {
		t.Errorf("Name() should return 'local', got %s", cache.Name())
	}
}

func TestLocalCache_TTLExpired(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latex-cache-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := LocalConfig{
		Dir: tmpDir,
		TTL: 1 * time.Millisecond, // 设置很短的 TTL
	}

	cache, err := NewLocalCache(cfg)
	if err != nil {
		t.Fatalf("创建缓存失败: %v", err)
	}

	ctx := context.Background()
	key := "test/expired.png"

	// 设置数据
	err = cache.Set(ctx, key, []byte("data"), 1*time.Millisecond)
	if err != nil {
		t.Fatalf("Set 失败: %v", err)
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	// 验证已过期
	result, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if result != nil {
		t.Errorf("Get expired key should return nil")
	}

	// 验证文件已删除
	path := filepath.Join(tmpDir, key)
	if _, err := os.Stat(path); err == nil {
		t.Errorf("Expired cache file should be deleted")
	}
}
