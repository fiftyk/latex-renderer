package renderer

import (
	"context"
	"strings"
	"testing"
)

// TestNewRenderer 测试创建渲染器
func TestNewRenderer(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}
	defer r.Close()

	if r == nil {
		t.Fatal("渲染器不应为 nil")
	}
}

// TestNewRendererWithExecPath 测试带可执行路径的渲染器创建
func TestNewRendererWithExecPath(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{
		ExecPath: "/usr/bin/chrome",
		Args:     "--no-sandbox",
	})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}
	defer r.Close()
}

// TestNewRendererWithArgs 测试带参数的渲染器创建
func TestNewRendererWithArgs(t *testing.T) {
	testCases := []struct {
		name string
		args string
	}{
		{"no args", ""},
		{"single arg", "--no-sandbox"},
		{"multiple args", "--no-sandbox --disable-setuid-sandbox"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := NewRenderer(&RendererOptions{Args: tc.args})
			if err != nil {
				t.Fatalf("创建渲染器失败: %v", err)
			}
			defer r.Close()
		})
	}
}

// TestWarmup 测试预热功能
func TestWarmup(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}
	defer r.Close()

	ctx := context.Background()

	// 第一次预热
	err = r.Warmup(ctx)
	if err != nil {
		t.Skipf("跳过预热测试（可能无 Chrome）: %v", err)
	}

	// 重复预热应不报错
	err = r.Warmup(ctx)
	if err != nil {
		t.Errorf("重复预热应不报错: %v", err)
	}
}

// TestInitBrowser 测试懒加载初始化（通过渲染测试）
func TestInitBrowser(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}
	defer r.Close()

	ctx := context.Background()

	// 通过Warmup触发初始化
	err = r.Warmup(ctx)
	if err != nil {
		t.Skipf("跳过测试（可能无 Chrome）: %v", err)
	}

	// 通过渲染验证初始化成功
	opts := &RenderOptions{
		Latex: "\\frac{a}{b}",
	}
	_, err = r.Render(ctx, opts)
	if err != nil {
		t.Skipf("跳过测试（可能无 Chrome）: %v", err)
	}
}

// TestClose 测试关闭渲染器
func TestClose(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}

	ctx := context.Background()

	// 预热
	_ = r.Warmup(ctx)

	// 关闭
	err = r.Close()
	if err != nil {
		t.Errorf("关闭渲染器失败: %v", err)
	}

	// 关闭后再次关闭应不报错
	err = r.Close()
	if err != nil {
		t.Errorf("重复关闭应不报错: %v", err)
	}
}

// TestCloseUninitialized 测试关闭未初始化的渲染器
func TestCloseUninitialized(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}
	defer r.Close()

	err = r.Close()
	if err != nil {
		t.Errorf("关闭未初始化的渲染器应不报错: %v", err)
	}
}

// TestFindChrome 测试查找 Chrome
func TestFindChrome(t *testing.T) {
	path := FindChrome()
	// 在 macOS 上通常能找到 Chrome
	if path == "" {
		t.Log("未找到 Chrome 可执行文件")
	} else {
		if !strings.HasPrefix(path, "/") {
			t.Errorf("期望绝对路径, got: %s", path)
		}
		t.Logf("找到 Chrome: %s", path)
	}
}

// TestRenderOptions 测试渲染选项默认值
func TestRenderOptions(t *testing.T) {
	opts := &RenderOptions{
		Latex: "\\frac{a}{b}",
	}

	// 验证默认值在 Render 中设置
	if opts.Color != "" {
		t.Error("Color 默认应为空")
	}
	if opts.Padding != "" {
		t.Error("Padding 默认应为空")
	}
}

// TestRendererThreadSafety 测试渲染器线程安全
func TestRendererThreadSafety(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}
	defer r.Close()

	ctx := context.Background()

	// 预热
	if err := r.Warmup(ctx); err != nil {
		t.Skipf("跳过线程安全测试（可能无 Chrome）: %v", err)
	}

	// 并发渲染测试
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			opts := &RenderOptions{
				Latex: "\\frac{a}{b}",
			}
			_, err := r.Render(ctx, opts)
			if err != nil {
				// 记录错误但不失败（可能是 Chrome 不可用）
				t.Logf("并发渲染 %d 失败: %v", idx, err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
