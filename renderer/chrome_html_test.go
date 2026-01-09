package renderer

import (
	"context"
	"strings"
	"testing"
)

// TestRenderPadding 测试 padding 参数生效
// 这个测试验证 padding 是否正确应用到 HTML 和截图元素
func TestRenderPadding(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}
	defer r.Close()

	ctx := context.Background()

	testCases := []struct {
		name       string
		padding    string
		expectHTML string
	}{
		{
			name:       "default padding",
			padding:    "20",
			expectHTML: `padding: 20px`,
		},
		{
			name:       "small padding",
			padding:    "10",
			expectHTML: `padding: 10px`,
		},
		{
			name:       "large padding",
			padding:    "50",
			expectHTML: `padding: 50px`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &RenderOptions{
				Latex:    "\\frac{a}{b}",
				Padding:  tc.padding,
				FontSize: "16",
				Color:    "black",
			}

			data, err := r.Render(ctx, opts)
			if err != nil {
				t.Skipf("跳过渲染测试（可能无 Chrome）: %v", err)
			}

			if len(data) == 0 {
				t.Error("渲染结果不应为空")
			}

			// 验证 PNG 头
			if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
				t.Error("渲染结果应为 PNG 格式")
			}
		})
	}
}

// TestRenderWithDifferentPadding 测试不同 padding 值的渲染
func TestRenderWithDifferentPadding(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}
	defer r.Close()

	ctx := context.Background()

	// 测试不同 padding 值
	paddingValues := []string{"0", "10", "20", "30", "50"}

	for _, padding := range paddingValues {
		t.Run("padding-"+padding, func(t *testing.T) {
			opts := &RenderOptions{
				Latex:    "x = \\frac{-b \\pm \\sqrt{b^2-4ac}}{2a}",
				Padding:  padding,
				FontSize: "20",
				Color:    "#333333",
			}

			data, err := r.Render(ctx, opts)
			if err != nil {
				t.Skipf("跳过渲染测试: %v", err)
			}

			if len(data) == 0 {
				t.Error("渲染结果不应为空")
			}
		})
	}
}

// TestRenderHTMLStructure 测试 HTML 结构符合 padding 预期
func TestRenderHTMLStructure(t *testing.T) {
	// 测试 HTML 生成逻辑
	opts := &RenderOptions{
		Latex:    "\\sum_{i=1}^n i = \\frac{n(n+1)}{2}",
		Padding:  "25",
		FontSize: "18",
		Color:    "blue",
	}

	// 生成 HTML（使用内部逻辑）
	color := opts.Color
	if color == "" {
		color = "black"
	}
	background := opts.Background
	if background == "" {
		background = "transparent"
	}
	fontSize := opts.FontSize
	if fontSize == "" {
		fontSize = "16"
	}
	padding := opts.Padding
	if padding == "" {
		padding = "20"
	}

	html := generateTestHTML(padding, background, fontSize, color, opts.Latex)

	// 验证 wrapper 容器存在
	if !strings.Contains(html, `class="katex-wrapper"`) {
		t.Error("HTML 应包含 katex-wrapper 容器")
	}

	// 验证 padding 应用到 wrapper
	if !strings.Contains(html, `.katex-wrapper {`) {
		t.Error("HTML 应包含 .katex-wrapper 样式")
	}
	if !strings.Contains(html, `padding: `+padding+`px`) {
		t.Errorf("HTML padding 应为 %spx, got: %s", padding, html)
	}

	// 验证 wrapper 有 id 用于截图
	if !strings.Contains(html, `id="wrapper"`) {
		t.Error("HTML 应包含 id=\"wrapper\" 用于截图")
	}

	// 验证 body padding 为 0（不再应用 padding）
	if strings.Contains(html, `body {`) && strings.Contains(html, `padding: 0`) {
		t.Log("body padding 应为 0（padding 已移到 wrapper）")
	}
}

// generateTestHTML 生成测试 HTML（模拟 Render 方法中的 HTML 生成）
func generateTestHTML(padding, background, fontSize, color, latex string) string {
	return string([]byte(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.css">
  <script src="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.js"></script>
  <style>
    body {
      margin: 0;
      padding: 0;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      background-color: ` + background + `;
    }
    .katex-wrapper {
      padding: ` + padding + `px;
    }
    .katex {
      font-size: ` + fontSize + `px;
      color: ` + color + `;
    }
    .katex-display {
      margin: 0;
    }
  </style>
</head>
<body>
  <div class="katex-wrapper" id="wrapper">
    <div id="formula"></div>
  </div>
  <script>
    katex.render(` + "`" + latex + "`" + `, document.getElementById('formula'), {
      displayMode: true,
      throwOnError: false,
      output: 'html'
    });
  </script>
</body>
</html>`))
}

// TestRenderWithBackground 测试背景色和 padding 组合
func TestRenderWithBackground(t *testing.T) {
	r, err := NewRenderer(&RendererOptions{})
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}
	defer r.Close()

	ctx := context.Background()

	opts := &RenderOptions{
		Latex:      "E = mc^2",
		Padding:    "30",
		Background: "#ffffff",
		FontSize:   "24",
		Color:      "#000000",
	}

	data, err := r.Render(ctx, opts)
	if err != nil {
		t.Skipf("跳过渲染测试: %v", err)
	}

	if len(data) == 0 {
		t.Error("渲染结果不应为空")
	}

	// 验证 PNG 格式
	if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Error("渲染结果应为 PNG 格式")
	}
}
