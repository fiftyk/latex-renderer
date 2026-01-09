package renderer

import "fmt"

// HTML模板生成器
type htmlGenerator struct {
	cssURL string
	jsURL  string
}

// NewHTMLGenerator 创建HTML生成器
func NewHTMLGenerator(cssURL, jsURL string) *htmlGenerator {
	return &htmlGenerator{
		cssURL: cssURL,
		jsURL:  jsURL,
	}
}

// GenerateHTML 生成渲染HTML
func (hg *htmlGenerator) GenerateHTML(latex, background, padding, fontSize, color string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="%s">
  <script src="%s"></script>
  <style>
    body {
      margin: 0;
      padding: 0;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      background-color: %s;
    }
    .katex-wrapper {
      padding: %spx;
    }
    .katex {
      font-size: %spx;
      color: %s;
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
    katex.render(%q, document.getElementById('formula'), {
      displayMode: true,
      throwOnError: false,
      output: 'html'
    });
  </script>
</body>
</html>`, hg.cssURL, hg.jsURL, background, padding, fontSize, color, latex)
}
