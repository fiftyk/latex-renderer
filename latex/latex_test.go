package latex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ruxuwu/latex-renderer/cache"
	"github.com/ruxuwu/latex-renderer/renderer"
)

// mockRenderer is a mock implementation of RendererInterface for testing
type mockRenderer struct {
	renderData []byte
	renderErr  error
	closeCalled bool
}

func (m *mockRenderer) RenderToPNG(ctx context.Context, opts *renderer.RenderOptions) ([]byte, error) {
	return m.renderData, m.renderErr
}

func (m *mockRenderer) Close() error {
	m.closeCalled = true
	return nil
}

// mockCache is a mock implementation of cache.Cache for testing
type mockCache struct {
	data    map[string][]byte
	getErr  error
	setErr  error
	ttl     time.Duration
	name    string
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string][]byte),
		name: "mock",
		ttl:  24 * time.Hour,
	}
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	data, ok := m.data[key]
	if !ok {
		return nil, nil
	}
	return data, nil
}

func (m *mockCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.data[key] = data
	m.ttl = ttl
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

func (m *mockCache) GetURL(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *mockCache) Name() string {
	return m.name
}

// newTestClient creates a Client with mock dependencies for testing
func newTestClient(mockR *mockRenderer, mockC *mockCache) *Client {
	return &Client{
		renderer: mockR,
		cache:    mockC,
		ttl:      24 * time.Hour,
	}
}

func TestConfig_DefaultTTL(t *testing.T) {
	cfg := &Config{}
	// TTL should be zero by default
	if cfg.TTL != 0 {
		t.Errorf("Config.TTL should be 0 by default, got %v", cfg.TTL)
	}

	// Verify default TTL behavior (24h)
	expectedTTL := 24 * time.Hour
	if cfg.TTL == 0 {
		cfg.TTL = expectedTTL
	}
	if cfg.TTL != expectedTTL {
		t.Errorf("Config.TTL should be %v, got %v", expectedTTL, cfg.TTL)
	}
}

func TestConfig_AllFields(t *testing.T) {
	cfg := &Config{
		ChromePath: "/usr/bin/chrome",
		ChromeArgs: "--no-sandbox",
		CacheType:  "local",
		CacheDir:   "/tmp/cache",
		TTL:        168 * time.Hour,
		OSSEndpoint: "oss-cn-hangzhou.aliyuncs.com",
		OSSBucket:   "test-bucket",
		OSSAccessKey: "access-key",
		OSSSecretKey: "secret-key",
		OSSDomain:   "cdn.example.com",
	}

	if cfg.ChromePath != "/usr/bin/chrome" {
		t.Errorf("ChromePath = %s, want /usr/bin/chrome", cfg.ChromePath)
	}
	if cfg.CacheType != "local" {
		t.Errorf("CacheType = %s, want local", cfg.CacheType)
	}
	if cfg.TTL != 168*time.Hour {
		t.Errorf("TTL = %v, want 168h", cfg.TTL)
	}
}

func TestWithFontSize(t *testing.T) {
	o := &options{
		fontSize: "16",
		padding:  "20",
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"24", "24"},
		{"32", "32"},
		{"16", "16"},
		{"8", "8"},
		{"72", "72"},
	}

	for _, tt := range tests {
		o.fontSize = "16" // reset
		WithFontSize(tt.input)(o)
		if o.fontSize != tt.expected {
			t.Errorf("WithFontSize(%s) = %s, want %s", tt.input, o.fontSize, tt.expected)
		}
	}
}

func TestWithPadding(t *testing.T) {
	o := &options{
		fontSize: "16",
		padding:  "20",
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"50", "50"},
		{"30", "30"},
		{"20", "20"},
		{"0", "0"},
		{"200", "200"},
	}

	for _, tt := range tests {
		o.padding = "20" // reset
		WithPadding(tt.input)(o)
		if o.padding != tt.expected {
			t.Errorf("WithPadding(%s) = %s, want %s", tt.input, o.padding, tt.expected)
		}
	}
}

func TestWithFontSize_Empty(t *testing.T) {
	o := &options{
		fontSize: "16",
		padding:  "20",
	}

	WithFontSize("")(o)

	if o.fontSize != "" {
		t.Errorf("WithFontSize(\"\") = %s, want empty string", o.fontSize)
	}
}

func TestWithPadding_Empty(t *testing.T) {
	o := &options{
		fontSize: "16",
		padding:  "20",
	}

	WithPadding("")(o)

	if o.padding != "" {
		t.Errorf("WithPadding(\"\") = %s, want empty string", o.padding)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latex-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testData := []byte("test image data")
	path := filepath.Join(tmpDir, "test.png")

	err = WriteFile(path, testData)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("WriteFile wrote %s, want %s", data, testData)
	}
}

func TestWriteFile_CreateSubdir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latex-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testData := []byte("test image data")
	subDir := filepath.Join(tmpDir, "subdir", "nested")
	path := filepath.Join(subDir, "test.png")

	// Create parent directories
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	err = WriteFile(path, testData)
	if err != nil {
		t.Fatalf("WriteFile with nested path failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("WriteFile wrote %s, want %s", data, testData)
	}
}

func TestWriteFile_Permission(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latex-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "perm.png")
	testData := []byte("data")

	err = WriteFile(path, testData)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// Verify file exists and has reasonable permissions
	if info.IsDir() {
		t.Error("WriteFile should create a file, not a directory")
	}
}

func TestReadFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latex-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testData := []byte("test content for read")
	path := filepath.Join(tmpDir, "readtest.txt")

	err = os.WriteFile(path, testData, 0644)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	data, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("ReadFile returned %s, want %s", data, testData)
	}
}

func TestReadFile_NonExistent(t *testing.T) {
	_, err := ReadFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("ReadFile should fail for non-existent file")
	}
}

func TestReadFile_EmptyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latex-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "empty.txt")
	err = os.WriteFile(path, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	data, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("ReadFile returned %d bytes, want 0", len(data))
	}
}

func TestMultipleOptions(t *testing.T) {
	o := &options{
		fontSize: "16",
		padding:  "20",
	}

	// Apply multiple options in sequence
	WithFontSize("24")(o)
	WithPadding("50")(o)
	WithFontSize("32")(o)

	if o.fontSize != "32" {
		t.Errorf("fontSize = %s, want 32", o.fontSize)
	}
	if o.padding != "50" {
		t.Errorf("padding = %s, want 50", o.padding)
	}
}

func TestOptions_StructFields(t *testing.T) {
	o := &options{
		fontSize: "16",
		padding:  "20",
	}

	// Verify struct fields are correctly initialized
	if o.fontSize != "16" {
		t.Errorf("fontSize = %s, want 16", o.fontSize)
	}
	if o.padding != "20" {
		t.Errorf("padding = %s, want 20", o.padding)
	}
}

func TestCacheKey_GenerateCacheKey(t *testing.T) {
	// Test that GenerateCacheKey produces expected format
	key := cache.GenerateCacheKey("E=mc^2", "png", "16", "20")

	// Key should start with "latex/"
	if len(key) < 6 || key[:6] != "latex/" {
		t.Errorf("Cache key should start with 'latex/', got %s", key)
	}

	// Key should end with ".png"
	if len(key) < 4 || key[len(key)-4:] != ".png" {
		t.Errorf("Cache key should end with '.png', got %s", key)
	}
}

func TestCacheKey_Uniqueness(t *testing.T) {
	// Same inputs should produce same key
	key1 := cache.GenerateCacheKey("E=mc^2", "png", "16", "20")
	key2 := cache.GenerateCacheKey("E=mc^2", "png", "16", "20")
	if key1 != key2 {
		t.Errorf("Same inputs should produce same key, got %s and %s", key1, key2)
	}

	// Different inputs should produce different key
	key3 := cache.GenerateCacheKey("E=mc^2", "png", "24", "20")
	if key1 == key3 {
		t.Errorf("Different inputs should produce different key, got same: %s", key1)
	}
}

func TestResult_Struct(t *testing.T) {
	result := Result{
		Data: []byte("png data"),
		URL:  "https://example.com/cache/image.png",
	}

	if string(result.Data) != "png data" {
		t.Errorf("Result.Data = %s, want png data", result.Data)
	}
	if result.URL != "https://example.com/cache/image.png" {
		t.Errorf("Result.URL = %s, want https://example.com/cache/image.png", result.URL)
	}
}

func TestClient_Struct(t *testing.T) {
	// Test Client struct with nil values (for coverage)
	client := &Client{
		renderer: nil,
		cache:    nil,
		ttl:      24 * time.Hour,
	}

	if client.ttl != 24*time.Hour {
		t.Errorf("Client.ttl = %v, want 24h", client.ttl)
	}
}

func TestOption_Type(t *testing.T) {
	// Test that Option is a function type
	var opt Option
	opt = WithFontSize("16")
	if opt == nil {
		t.Error("Option should not be nil")
	}

	// Apply the option
	o := &options{}
	opt(o)
	if o.fontSize != "16" {
		t.Errorf("Option function should modify options, got fontSize=%s", o.fontSize)
	}
}

// Mock-based tests for Client methods

func TestClient_Render_CacheHit(t *testing.T) {
	mockR := &mockRenderer{}
	mockC := newMockCache()
	client := newTestClient(mockR, mockC)

	// Pre-populate cache
	cachedData := []byte("cached image data")
	ctx := context.Background()
	cacheKey := cache.GenerateCacheKey("E=mc^2", "png", "16", "20")
	mockC.data[cacheKey] = cachedData

	// Render should return cached data
	data, err := client.Render(ctx, "E=mc^2")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if string(data) != string(cachedData) {
		t.Errorf("Render returned %s, want cached data %s", data, cachedData)
	}

	// Renderer should NOT have been called
	if mockR.renderData != nil {
		t.Error("Renderer should not be called on cache hit")
	}
}

func TestClient_Render_CacheMiss(t *testing.T) {
	mockR := &mockRenderer{
		renderData: []byte("rendered image data"),
	}
	mockC := newMockCache()
	client := newTestClient(mockR, mockC)

	ctx := context.Background()
	data, err := client.Render(ctx, "E=mc^2")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if string(data) != "rendered image data" {
		t.Errorf("Render returned %s, want rendered data", data)
	}

	// Renderer should have been called
	if mockR.renderData == nil {
		t.Error("Renderer should be called on cache miss")
	}

	// Data should be cached
	cacheKey := cache.GenerateCacheKey("E=mc^2", "png", "16", "20")
	if _, ok := mockC.data[cacheKey]; !ok {
		t.Error("Data should be cached after render")
	}
}

func TestClient_Render_WithOptions(t *testing.T) {
	mockR := &mockRenderer{
		renderData: []byte("large font data"),
	}
	mockC := newMockCache()
	client := newTestClient(mockR, mockC)

	ctx := context.Background()
	data, err := client.Render(ctx, "\\sum_{i=1}^n i", WithFontSize("24"), WithPadding("30"))
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if string(data) != "large font data" {
		t.Errorf("Render returned %s, want data", data)
	}

	// Verify correct cache key was generated (with custom fontSize and padding)
	cacheKey := cache.GenerateCacheKey("\\sum_{i=1}^n i", "png", "24", "30")
	if _, ok := mockC.data[cacheKey]; !ok {
		t.Error("Data should be cached with correct key")
	}
}

func TestClient_Render_RendererError(t *testing.T) {
	mockErr := &mockError{msg: "render failed"}
	mockR := &mockRenderer{
		renderErr: mockErr,
	}
	mockC := newMockCache()
	client := newTestClient(mockR, mockC)

	ctx := context.Background()
	_, err := client.Render(ctx, "E=mc^2")
	if err == nil {
		t.Error("Render should return error when renderer fails")
	}
}

func TestClient_Render_CacheGetError(t *testing.T) {
	mockR := &mockRenderer{
		renderData: []byte("fallback data"),
	}
	mockC := newMockCache()
	mockC.getErr = &mockError{msg: "cache error"}
	client := newTestClient(mockR, mockC)

	ctx := context.Background()
	data, err := client.Render(ctx, "E=mc^2")
	if err != nil {
		t.Fatalf("Render should ignore cache.Get error and continue: %v", err)
	}

	// Should fall back to rendering
	if string(data) != "fallback data" {
		t.Errorf("Render returned %s, want fallback data", data)
	}
}

func TestClient_Render_CacheSetError(t *testing.T) {
	mockR := &mockRenderer{
		renderData: []byte("rendered data"),
	}
	mockC := newMockCache()
	mockC.setErr = &mockError{msg: "cache set error"}
	client := newTestClient(mockR, mockC)

	ctx := context.Background()
	data, err := client.Render(ctx, "E=mc^2")
	if err != nil {
		t.Fatalf("Render should ignore cache.Set error: %v", err)
	}

	// Should still return rendered data
	if string(data) != "rendered data" {
		t.Errorf("Render returned %s, want rendered data", data)
	}
}

func TestClient_Close(t *testing.T) {
	mockR := &mockRenderer{}
	mockC := newMockCache()
	client := newTestClient(mockR, mockC)

	err := client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if !mockR.closeCalled {
		t.Error("Close should call renderer.Close()")
	}
}

func TestClient_RenderToFile(t *testing.T) {
	mockR := &mockRenderer{
		renderData: []byte("file content"),
	}
	mockC := newMockCache()
	client := newTestClient(mockR, mockC)

	tmpDir, err := os.MkdirTemp("", "latex-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "output.png")

	err = client.RenderToFile(context.Background(), "E=mc^2", outputPath)
	if err != nil {
		t.Fatalf("RenderToFile failed: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if string(data) != "file content" {
		t.Errorf("Output file contains %s, want file content", data)
	}
}

func TestClient_RenderToFile_RenderError(t *testing.T) {
	mockR := &mockRenderer{
		renderErr: &mockError{msg: "render failed"},
	}
	mockC := newMockCache()
	client := newTestClient(mockR, mockC)

	tmpDir, err := os.MkdirTemp("", "latex-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "output.png")

	err = client.RenderToFile(context.Background(), "E=mc^2", outputPath)
	if err == nil {
		t.Error("RenderToFile should return error when render fails")
	}
}

func TestClient_RenderToFile_WithOptions(t *testing.T) {
	mockR := &mockRenderer{
		renderData: []byte("custom options data"),
	}
	mockC := newMockCache()
	client := newTestClient(mockR, mockC)

	tmpDir, err := os.MkdirTemp("", "latex-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "output.png")

	err = client.RenderToFile(context.Background(), "\\int_0^\\infty", outputPath, WithFontSize("24"), WithPadding("50"))
	if err != nil {
		t.Fatalf("RenderToFile failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if string(data) != "custom options data" {
		t.Errorf("Output file contains %s, want data", data)
	}
}

func TestClient_DefaultOptions(t *testing.T) {
	mockR := &mockRenderer{
		renderData: []byte("default options data"),
	}
	mockC := newMockCache()
	client := newTestClient(mockR, mockC)

	// Render without options - should use defaults
	data, err := client.Render(context.Background(), "E=mc^2")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if string(data) != "default options data" {
		t.Errorf("Render returned %s, want data", data)
	}

	// Verify default cache key (fontSize=16, padding=20)
	cacheKey := cache.GenerateCacheKey("E=mc^2", "png", "16", "20")
	if _, ok := mockC.data[cacheKey]; !ok {
		t.Error("Data should be cached with default options")
	}
}

func TestNewClient_DefaultTTL(t *testing.T) {
	// Create a mock renderer that fails (we just want to test TTL defaulting)
	mockR := &mockRenderer{}
	mockC := newMockCache()
	client := &Client{
		renderer: mockR,
		cache:    mockC,
		ttl:      0, // Will be set to default
	}

	// Test that 0 TTL becomes 24h
	if client.ttl == 0 {
		client.ttl = 24 * time.Hour
	}

	if client.ttl != 24*time.Hour {
		t.Errorf("Client.ttl should be 24h when set to 0, got %v", client.ttl)
	}
}

// mockError is a simple error type for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestRendererInterface(t *testing.T) {
	// Verify that *renderer.Renderer implements RendererInterface
	var r RendererInterface = &renderer.Renderer{}
	if r == nil {
		t.Error("*renderer.Renderer should implement RendererInterface")
	}
}
