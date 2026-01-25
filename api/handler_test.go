package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// MockStrategy 模拟过载策略用于测试
type MockStrategy struct {
	shouldAccept bool
}

func (m *MockStrategy) Handle() bool {
	return m.shouldAccept
}

func (m *MockStrategy) Release() {}

func (m *MockStrategy) Reject() error {
	return nil
}

// MockCache 模拟缓存用于测试
type MockCache struct {
	data map[string][]byte
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string][]byte),
	}
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if data, ok := m.data[key]; ok {
		return data, nil
	}
	return nil, nil
}

func (m *MockCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	m.data[key] = data
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

func (m *MockCache) GetURL(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *MockCache) Name() string {
	return "mock"
}

func TestRender_MissingLatex(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// 创建 MockCache
	mockCache := NewMockCache()

	// 创建处理器
	handler := &Handler{
		cache:            mockCache,
		ttl:              24 * time.Hour,
		overloadStrategy: &MockStrategy{shouldAccept: true},
	}

	router.GET("/api", handler.Render)

	// 测试缺少 latex 参数
	req, _ := http.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp RenderResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Message != "缺少 latex 参数" {
		t.Errorf("Expected error message '缺少 latex 参数', got %s", resp.Message)
	}
}

func TestRender_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	mockCache := NewMockCache()

	handler := &Handler{
		cache:            mockCache,
		ttl:              24 * time.Hour,
		overloadStrategy: &MockStrategy{shouldAccept: true},
	}

	router.GET("/api", handler.Render)

	// 测试不支持的格式
	req, _ := http.NewRequest("GET", "/api?latex=\\frac{a}{b}&format=webp", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp RenderResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Success {
		t.Error("Expected success to be false")
	}
}

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	mockCache := NewMockCache()

	handler := &Handler{
		cache:            mockCache,
		ttl:              24 * time.Hour,
		overloadStrategy: &MockStrategy{shouldAccept: true},
	}

	router.GET("/health", handler.Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", resp["status"])
	}
	if resp["cache"] != "mock" {
		t.Errorf("Expected cache 'mock', got %v", resp["cache"])
	}
}

func TestInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	mockCache := NewMockCache()

	handler := &Handler{
		cache:            mockCache,
		ttl:              24 * time.Hour,
		overloadStrategy: &MockStrategy{shouldAccept: true},
	}

	router.GET("/info", handler.Info)

	req, _ := http.NewRequest("GET", "/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["name"] != "latex-renderer" {
		t.Errorf("Expected name 'latex-renderer', got %v", resp["name"])
	}
	if resp["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", resp["version"])
	}
	if resp["cache"] != "mock" {
		t.Errorf("Expected cache 'mock', got %v", resp["cache"])
	}
}

func TestSetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	mockCache := NewMockCache()

	handler := &Handler{
		cache:            mockCache,
		ttl:              24 * time.Hour,
		overloadStrategy: &MockStrategy{shouldAccept: true},
	}

	SetupRoutes(router, handler)

	// 测试路由是否正确设置
	routes := router.Routes()
	expectedRoutes := map[string]string{
		"/health": "GET",
		"/info":   "GET",
		"/api":    "GET",
	}

	for _, route := range routes {
		if expectedMethod, ok := expectedRoutes[route.Path]; ok {
			if route.Method != expectedMethod {
				t.Errorf("Route %s expected method %s, got %s", route.Path, expectedMethod, route.Method)
			}
		}
	}
}
